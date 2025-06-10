package postgres

import (
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"gorm.io/gorm"
)

func toDomainUser(m *userModel) *user.User {
	return &user.User{
		ID:         m.ID,
		TelegramID: m.TelegramID,
		LastMenu:   m.LastMenu,
		IsAdmin:    m.IsAdmin,
		CreatedAt:  m.CreatedAt,
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

func fromDomainUser(d *user.User) *userModel {
	return &userModel{
		Model:      gorm.Model{ID: d.ID, CreatedAt: d.CreatedAt},
		TelegramID: d.TelegramID,
		LastMenu:   d.LastMenu,
		IsAdmin:    d.IsAdmin,
		Profile: profileModel{
			Model:     gorm.Model{ID: d.Profile.ID},
			UserID:    d.Profile.UserID,
			Username:  d.Profile.Username,
			FirstName: d.Profile.FirstName,
			LastName:  d.Profile.LastName,
			Score:     d.Profile.Score,
		},
	}
}
