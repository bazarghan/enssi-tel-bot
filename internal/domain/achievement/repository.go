package achievement

import "context"

// Repository defines the persistence port for achievement data.
type Repository interface {

	// -------- Find Method ----------------------
	FindAll(ctx context.Context) ([]Achievement, error)
	FindByID(ctx context.Context, achievementID uint) (Achievement, error)
	FindByTitle(ctx context.Context, title string) (Achievement, error)
	FindUserAchievements(ctx context.Context, userID uint) ([]UserAchievement, error)
	FindAllProgressiveDaily(ctx context.Context) ([]Achievement, error)

	// --------- Get and save operation -----------
	GetUserAchievement(ctx context.Context, userID, achievementID uint) (UserAchievement, error)
	SaveUserAchievement(ctx context.Context, ua UserAchievement, totalItems uint) error
}
