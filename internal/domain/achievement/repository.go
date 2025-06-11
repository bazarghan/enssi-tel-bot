package achievement

import "context"

// Repository defines the persistence port for achievement data.
type Repository interface {
	FindAll(ctx context.Context) ([]Achievement, error)
	FindByID(ctx context.Context, achievementID uint) (Achievement, error)
	GetUserAchievement(ctx context.Context, userID, achievementID uint) (UserAchievement, error)
	SaveUserAchievement(ctx context.Context, ua UserAchievement) error
}
