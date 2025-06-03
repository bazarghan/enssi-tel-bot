package user

import (
	"github.com/2000ostd/enssi-tel-bot/internal/models"
)

type UserService interface {
	GetOrCreateUserByTelegramID(telegramID int64, username string, firstName string, lastName string) (*models.User, error)
	UpdateUserLastMenu(userID uint, menuState string) error
	GetUserProfile(userID uint) (*UserProfileView, error)
	UpdateUserProfile(userID uint, req UpdateProfileRequest) error
	RecordUserActivity(userID uint) error
}
