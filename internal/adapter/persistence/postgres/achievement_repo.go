package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/achievement"
	"gorm.io/gorm"
)

// AchievementRepository is the GORM implementation of the repository port.
type AchievementRepository struct {
	db *gorm.DB
}

// NewAchievementRepository creates a new repository.
func NewAchievementRepository(db *gorm.DB) *AchievementRepository {
	return &AchievementRepository{db: db}
}

// FindAll retrieves all defined achievements.
func (r *AchievementRepository) FindAll(ctx context.Context) ([]achievement.Achievement, error) {
	var models []achievementModel
	if err := r.db.WithContext(ctx).Order("id asc").Find(&models).Error; err != nil {
		return nil, err
	}

	achievements := make([]achievement.Achievement, len(models))
	for i, m := range models {
		achievements[i] = toDomainAchievement(m)
	}
	return achievements, nil
}

// FindByID retrieves a single achievement definition.
func (r *AchievementRepository) FindByID(ctx context.Context, achievementID uint) (achievement.Achievement, error) {
	var model achievementModel
	if err := r.db.WithContext(ctx).First(&model, achievementID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return achievement.Achievement{}, achievement.ErrNotFound
		}
		return achievement.Achievement{}, err
	}
	return toDomainAchievement(model), nil
}

// GetUserAchievement retrieves a user's specific progress on an achievement.
func (r *AchievementRepository) GetUserAchievement(ctx context.Context, userID, achievementID uint) (achievement.UserAchievement, error) {
	var model userAchievementModel

	err := r.db.WithContext(ctx).
		Joins("JOIN profiles ON profiles.id = profile_achievements.profile_id").
		Where("profiles.user_id = ? AND profile_achievements.achievement_id = ?", userID, achievementID).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return achievement.UserAchievement{}, achievement.ErrUserAchNotFound
		}
		return achievement.UserAchievement{}, err
	}
	return toDomainUserAchievement(model), nil
}

// SaveUserAchievement creates or updates a user's achievement progress.
func (r *AchievementRepository) SaveUserAchievement(ctx context.Context, ua achievement.UserAchievement) error {
	// 1. Find the profile_id for the given user_id
	var profileID uint
	err := r.db.WithContext(ctx).Model(&profileModel{}).Select("id").Where("user_id = ?", ua.UserID).Row().Scan(&profileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("could not find profile for user_id %d to save achievement", ua.UserID)
		}
		return err
	}
	if profileID == 0 {
		return fmt.Errorf("found zero value for profile_id for user_id %d", ua.UserID)
	}

	model := toPersistenceUserAchievement(ua, profileID)
	// GORM's Save method handles both INSERT (if ID is 0) and UPDATE.
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return achievement.ErrSaveFailed
	}
	return nil
}

// FindUserAchievements retrieves all of a user's achievement progress records, preloading the base achievement data.
func (r *AchievementRepository) FindUserAchievements(ctx context.Context, userID uint) ([]achievement.UserAchievement, error) {
	var models []userAchievementModel

	err := r.db.WithContext(ctx).
		Joins("Achievement"). // Preload the associated Achievement details
		Joins("JOIN profiles ON profiles.id = profile_achievements.profile_id").
		Where("profiles.user_id = ?", userID).
		Find(&models).Error

	if err != nil {
		return nil, err
	}

	userAchievements := make([]achievement.UserAchievement, len(models))
	for i, m := range models {
		userAchievements[i] = toDomainUserAchievement(m)
		userAchievements[i].Details = toDomainAchievement(m.Achievement) // Manually map the preloaded details
	}
	return userAchievements, nil
}
