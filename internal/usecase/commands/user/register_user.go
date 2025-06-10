// internal/usecase/commands/user/register_user.go
package user

import (
	"context"
	"errors"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
)

// Command holds the input data for registering a user.
type Command struct {
	TelegramID int64
	Username   string
	FirstName  string
	LastName   string
}

// Handler orchestrates the user registration process.
type Handler struct {
	userRepo user.Repository
}

func NewHandler(userRepo user.Repository) *Handler {
	return &Handler{userRepo: userRepo}
}

// Handle processes the user registration command.
func (h *Handler) Handle(ctx context.Context, cmd Command) (*user.User, error) {
	// 1. Try to get the user
	existingUser, err := h.userRepo.GetByTelegramID(ctx, cmd.TelegramID)
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		return nil, err // A real error occurred
	}

	// 2. User exists, update their profile info if changed
	if existingUser != nil {
		profileChanged := false
		if existingUser.Profile.Username != cmd.Username {
			existingUser.Profile.Username = cmd.Username
			profileChanged = true
		}
		// ... check other fields (FirstName, LastName) ...

		if profileChanged {
			if err := h.userRepo.Save(ctx, existingUser); err != nil {
				return nil, err
			}
		}
		return existingUser, nil
	}

	// 3. User does not exist, create a new one
	newUser := &user.User{
		TelegramID: cmd.TelegramID,
		LastMenu:   "main",
		IsAdmin:    false,
		Profile: user.Profile{
			Username:  cmd.Username,
			FirstName: cmd.FirstName,
			LastName:  cmd.LastName,
			Score:     0,
		},
	}

	if err := h.userRepo.Save(ctx, newUser); err != nil {
		return nil, err
	}

	// After saving, the newUser entity might have the DB-generated ID.
	// A robust implementation re-fetches the user to get the full entity.
	return h.userRepo.GetByTelegramID(ctx, cmd.TelegramID)
}
