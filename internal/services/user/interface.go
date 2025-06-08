package user

import (
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"github.com/bits-and-blooms/bitset" // Import for bitset
)

type UserService interface {
	GetOrCreateUserByTelegramID(telegramID int64, username string, firstName string, lastName string) (*models.User, error)
	UpdateUserLastMenu(userID uint, menuState string) error
	GetUserProfile(userID uint) (*UserProfileView, error)
	UpdateUserProfile(userID uint, req UpdateProfileRequest) error
	RecordUserActivity(userID uint) error
	MarkReviewSessionCompleted(userID uint, completedAt time.Time) error

	AwardAchievementProgress(userID uint, achievementID uint, itemsToReveal int) (*models.ProfileAchievement, *bitset.BitSet, error)

	CompleteAchievement(userID uint, achievementID uint) (*models.ProfileAchievement, *bitset.BitSet, error)
}
