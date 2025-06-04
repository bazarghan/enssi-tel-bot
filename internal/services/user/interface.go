package user

import (
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/models"
)

type UserService interface {
	GetOrCreateUserByTelegramID(telegramID int64, username string, firstName string, lastName string) (*models.User, error)
	UpdateUserLastMenu(userID uint, menuState string) error
	GetUserProfile(userID uint) (*UserProfileView, error)
	UpdateUserProfile(userID uint, req UpdateProfileRequest) error
	RecordUserActivity(userID uint) error

	// MarkReviewSessionCompleted updates the user's timestamp for their last completed review session.
	MarkReviewSessionCompleted(userID uint, completedAt time.Time) error
}
