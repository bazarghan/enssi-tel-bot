package user

import (
	"context"
	"fmt"
	"github.com/bazarghan/enssi-tel-bot/internal/domain/user"
)

// RegisterUserCommand defines the input for registering a user.
type RegisterUserCommand struct {
	TelegramID int64
	Username   string
	FirstName  string
	LastName   string
}

// RegisterUserResult defines the output.
type RegisterUserResult struct {
	User user.User
}

// RegisterUserHandler processes the registration command.
type RegisterUserHandler struct {
	userRepo user.Repository
}

// NewRegisterUserHandler creates a new handler.
func NewRegisterUserHandler(userRepo user.Repository) RegisterUserHandler {
	return RegisterUserHandler{userRepo: userRepo}
}

// Handle executes the command by calling the repository to either get or create the user.
func (h RegisterUserHandler) Handle(ctx context.Context, cmd RegisterUserCommand) (RegisterUserResult, error) {
	domainUser, err := h.userRepo.GetOrCreate(ctx, cmd.TelegramID, cmd.Username, cmd.FirstName, cmd.LastName)
	if err != nil {
		return RegisterUserResult{}, fmt.Errorf("failed to get or create user: %w", err)
	}

	return RegisterUserResult{User: domainUser}, nil
}

