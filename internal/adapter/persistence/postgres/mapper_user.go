package postgres

import (
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"gorm.io/gorm"
	"time"
)

// toDomainUser converts a GORM userModel to a domain User entity.
func toDomainUser(m UserModel) user.User {
	return user.User{
		ID:                           m.ID,
		TelegramID:                   m.TelegramID,
		IsAdmin:                      m.IsAdmin,
		LastMenu:                     m.LastMenu,
		LastReviewSessionCompletedAt: m.LastReviewSessionCompletedAt,
		Profile: user.Profile{
			ID:        m.Profile.ID,
			UserID:    m.Profile.UserID,
			Username:  m.Profile.Username,
			FirstName: m.Profile.FirstName,
			LastName:  m.Profile.LastName,
			Score:     m.Profile.Score,
		},
	}
}

// toPersistenceUserModel converts a domain User entity back to a GORM userModel for saving.
func toPersistenceUserModel(u user.User) UserModel {
	return UserModel{
		Model:                        gorm.Model{ID: u.ID, CreatedAt: time.Time{}, UpdatedAt: time.Time{}},
		TelegramID:                   u.TelegramID,
		IsAdmin:                      u.IsAdmin,
		LastMenu:                     u.LastMenu,
		LastActive:                   time.Now(), // Update activity on save
		LastOnline:                   time.Now(),
		LastReviewSessionCompletedAt: u.LastReviewSessionCompletedAt,
	}
}

// toPersistenceProfileModel converts a domain Profile entity to a GORM profileModel for saving.
func toPersistenceProfileModel(p user.Profile) ProfileModel {
	return ProfileModel{
		Model:     gorm.Model{ID: p.ID, CreatedAt: time.Time{}, UpdatedAt: time.Time{}},
		UserID:    p.UserID,
		Username:  p.Username,
		FirstName: p.FirstName,
		LastName:  p.LastName,
		Score:     p.Score,
	}
}
