package postgres

import (
	"context"
	"errors"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"gorm.io/gorm"
)

type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*user.User, error) {
	var model userModel
	err := r.db.WithContext(ctx).Preload("Profile").First(&model, "telegram_id = ?", telegramID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound // Use domain-specific error
		}
		return nil, err
	}
	return toDomainUser(&model), nil
}

func (r *GormUserRepository) Save(ctx context.Context, domainUser *user.User) error {
	gormUser := fromDomainUser(domainUser)
	// GORM's Save handles both creation and updates automatically based on the primary key (ID).
	return r.db.WithContext(ctx).Save(gormUser).Error
}
