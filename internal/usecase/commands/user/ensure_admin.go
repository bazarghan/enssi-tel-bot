package user

import (
	"context"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
)

// EnsureAdminCommand holds the data needed to ensure a user is an admin.
type EnsureAdminCommand struct {
	TelegramID int64
}

// EnsureAdminHandler is the use case for ensuring a user has admin privileges on startup.
type EnsureAdminHandler struct {
	logger   logger.Logger
	userRepo user.Repository
}

// NewEnsureAdminHandler creates a new handler.
func NewEnsureAdminHandler(appLogger logger.Logger, userRepo user.Repository) EnsureAdminHandler {
	return EnsureAdminHandler{
		logger:   appLogger,
		userRepo: userRepo,
	}
}

// Handle executes the command.
func (h EnsureAdminHandler) Handle(ctx context.Context, cmd EnsureAdminCommand) error {
	// If no admin ID is configured, there's nothing to do.
	if cmd.TelegramID == 0 {
		h.logger.Info("No admin_telegram_id configured, skipping admin promotion.")
		return nil
	}

	h.logger.Info("Ensuring admin status for user", "telegram_id", cmd.TelegramID)

	// 1. Get or Create the admin user. This ensures the record exists.
	// We pass empty strings for profile details as they aren't needed here.
	adminUser, err := h.userRepo.GetOrCreate(ctx, cmd.TelegramID, "", "", "")
	if err != nil {
		return fmt.Errorf("could not get or create admin user: %w", err)
	}

	// 2. If the user is already an admin, we don't need to do anything else.
	if adminUser.IsAdmin {
		h.logger.Info("User is already an admin.", "telegram_id", cmd.TelegramID)
		return nil
	}

	// 3. Promote the user and save the changes.
	h.logger.Info("User is not an admin, promoting...", "telegram_id", cmd.TelegramID)
	adminUser.IsAdmin = true

	if err := h.userRepo.Save(ctx, adminUser); err != nil {
		return fmt.Errorf("failed to save admin status for telegram_id %d: %w", cmd.TelegramID, err)
	}

	h.logger.Info("Successfully ensured admin status.", "telegram_id", cmd.TelegramID)
	return nil
}
