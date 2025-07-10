package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/bazarghan/enssi-tel-bot/internal/domain/user"
	"gorm.io/gorm"
	"time"
)

// UserRepository is the GORM implementation of the user repository port.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetOrCreate finds a user by Telegram ID or creates one.
func (r *UserRepository) GetOrCreate(ctx context.Context, telegramID int64, username, firstName, lastName string) (user.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).Where("telegram_id = ?", telegramID).Preload("Profile").First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new user
			return r.createUser(ctx, telegramID, username, firstName, lastName)
		}
		return user.User{}, fmt.Errorf("db error fetching user by telegram_id %d: %w", telegramID, err)
	}

	// Update existing user if needed
	return r.updateUser(ctx, model, username, firstName, lastName)
}

func (r *UserRepository) createUser(ctx context.Context, telegramID int64, username, firstName, lastName string) (user.User, error) {
	newUser := UserModel{
		TelegramID: telegramID,
		LastMenu:   "main",
		LastActive: time.Now(),
		LastOnline: time.Now(),
		Profile: ProfileModel{
			Username:  username,
			FirstName: firstName,
			LastName:  lastName,
		},
	}

	if err := r.db.WithContext(ctx).Create(&newUser).Error; err != nil {
		return user.User{}, fmt.Errorf("%w: %w", user.ErrCreateFailed, err)
	}
	return toDomainUser(newUser), nil
}

func (r *UserRepository) updateUser(ctx context.Context, model UserModel, username, firstName, lastName string) (user.User, error) {
	model.LastActive = time.Now()
	model.LastOnline = time.Now()

	profileChanged := false
	if model.Profile.Username != username {
		model.Profile.Username = username
		profileChanged = true
	}
	if model.Profile.FirstName != firstName {
		model.Profile.FirstName = firstName
		profileChanged = true
	}
	if model.Profile.LastName != lastName {
		model.Profile.LastName = lastName
		profileChanged = true
	}

	tx := r.db.WithContext(ctx).Begin()
	if err := tx.Save(&model).Error; err != nil {
		tx.Rollback()
		return user.User{}, fmt.Errorf("%w: could not save user model: %w", user.ErrSaveFailed, err)
	}
	if profileChanged {
		if err := tx.Save(&model.Profile).Error; err != nil {
			tx.Rollback()
			return user.User{}, fmt.Errorf("%w: could not save profile model: %w", user.ErrSaveFailed, err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		return user.User{}, fmt.Errorf("%w: transaction commit failed: %w", user.ErrSaveFailed, err)
	}

	return toDomainUser(model), nil
}

// FindByID retrieves a user by their internal database ID.
func (r *UserRepository) FindByID(ctx context.Context, userID uint) (user.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).Where("id = ?", userID).Preload("Profile").First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user.User{}, user.ErrNotFound
		}
		return user.User{}, err
	}
	return toDomainUser(model), nil
}

// UpdateLastMenu updates the user's last known menu state.
func (r *UserRepository) UpdateLastMenu(ctx context.Context, userID uint, menuState string) error {
	result := r.db.WithContext(ctx).Model(&UserModel{}).Where("id = ?", userID).Update("last_menu", menuState)
	if result.Error != nil {
		return fmt.Errorf("db error updating last_menu: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return user.ErrNotFound
	}
	return nil
}

// FindAllIDs retrieves all user IDs.
func (r *UserRepository) FindAllIDs(ctx context.Context) ([]uint, error) {
	var ids []uint
	err := r.db.WithContext(ctx).Model(&UserModel{}).Pluck("id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("could not pluck user ids: %w", err)
	}
	return ids, nil
}

// FindTelegramID retrieves a user's Telegram ID.
func (r *UserRepository) FindTelegramID(ctx context.Context, userID uint) (int64, error) {
	var model UserModel
	err := r.db.WithContext(ctx).Model(&UserModel{}).Where("id = ?", userID).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, user.ErrNotFound
		}
		return 0, err
	}
	return model.TelegramID, nil
}

func (r *UserRepository) UpdateLastReviewSession(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&UserModel{}).
		Where("id = ?", userID).
		Update("last_review_session_completed_at", time.Now()).Error
}

func (r *UserRepository) CountTotalUsers(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&UserModel{}).Count(&count).Error
	return count, err
}

// SetAdminStatus updates a user's admin flag based on their Telegram ID.
// It will not return an error if the user does not exist.
func (r *UserRepository) SetAdminStatus(ctx context.Context, telegramID int64, isAdmin bool) error {
	// This query finds the user by their telegram_id and updates only the is_admin column.
	result := r.db.WithContext(ctx).Model(&UserModel{}).
		Where("telegram_id = ?", telegramID).
		Update("is_admin", isAdmin)

	return result.Error
}

// Save persists all changes to a User object and its associated Profile.
func (r *UserRepository) Save(ctx context.Context, u user.User) error {
	userModel := toPersistenceUserModel(u)
	profileModel := toPersistenceProfileModel(u.Profile)

	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Save(&userModel).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Ensure the profile has the correct UserID before saving
	if profileModel.UserID == 0 {
		profileModel.UserID = userModel.ID
	}

	if err := tx.Save(&profileModel).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
