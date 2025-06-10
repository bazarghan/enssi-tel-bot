// internal/domain/user/repository.go
package user

import "context"

type Repository interface {
	// GetByTelegramID finds a user by their Telegram ID.
	GetByTelegramID(ctx context.Context, telegramID int64) (*User, error)
	// Save creates a new user or updates an existing one.
	Save(ctx context.Context, user *User) error
}
