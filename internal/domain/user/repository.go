package user

import "context"

// Repository defines the persistence port for user-related data.
// This interface is implemented by an adapter in an outer layer (e.g., persistence/postgres).
type Repository interface {
	// GetOrCreate finds a user by their Telegram ID. If the user does not exist,
	// it creates a new user and their profile, then returns the new user.
	// If the user exists, it updates their profile details if they have changed and returns the user.
	GetOrCreate(ctx context.Context, telegramID int64, username, firstName, lastName string) (User, error)

	// FindByID retrieves a user by their internal database ID.
	FindByID(ctx context.Context, userID uint) (User, error)

	// UpdateLastMenu updates the user's last known menu state.
	UpdateLastMenu(ctx context.Context, userID uint, menuState string) error
}

