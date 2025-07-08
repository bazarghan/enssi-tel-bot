package user

import "context"

type Repository interface {

	// GetOrCreate finds a user by their Telegram ID. If the user does not exist,
	GetOrCreate(ctx context.Context, telegramID int64, username, firstName, lastName string) (User, error)

	// FindByID retrieves a user by their internal database ID.
	FindByID(ctx context.Context, userID uint) (User, error)

	// UpdateLastMenu updates the user's last known menu state.
	UpdateLastMenu(ctx context.Context, userID uint, menuState string) error

	// FindAllIDs retrieves all user IDs for batch processing.
	FindAllIDs(ctx context.Context) ([]uint, error)

	// FindTelegramID retrieves a user's Telegram ID from their internal application ID.
	FindTelegramID(ctx context.Context, userID uint) (int64, error)

	// Update the Last Review session
	UpdateLastReviewSession(ctx context.Context, userID uint) error

	// Count the total number of users
	CountTotalUsers(ctx context.Context) (int64, error)

	// SetAdminStatus updates a user's admin flag based on their Telegram ID.
	SetAdminStatus(ctx context.Context, telegramID int64, isAdmin bool) error

	// Save persists all changes to a User and their Profile.
	Save(ctx context.Context, user User) error
}
