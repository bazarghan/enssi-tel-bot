package postgres

import (
	"context"
	"errors"
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
	// Note: The old schema linked to ProfileID. The new domain uses UserID.
	// This implementation assumes a direct UserID link for simplicity.
	// A join on profiles might be needed if the schema is more complex.
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND achievement_id = ?", userID, achievementID).
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
	model := toPersistenceUserAchievement(ua)
	// GORM's Save method handles both INSERT (if ID is 0) and UPDATE.
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return achievement.ErrSaveFailed
	}
	return nil
}
