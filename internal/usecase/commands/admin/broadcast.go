package admin

import (
	"context"
	"time"

	"github.com/bazarghan/enssi-tel-bot/internal/domain/notification"
	"github.com/bazarghan/enssi-tel-bot/internal/domain/user"
	"github.com/bazarghan/enssi-tel-bot/internal/platform/observability/logger"
)

// BroadcastCommand holds the message to be sent to all users.
type BroadcastCommand struct {
	Message string
}

// BroadcastHandler is responsible for sending a message to all users.
type BroadcastHandler struct {
	logger   logger.Logger
	userRepo user.Repository
	notifier notification.Notifier
}

// NewBroadcastHandler creates a new BroadcastHandler.
func NewBroadcastHandler(
	appLogger logger.Logger,
	userRepo user.Repository,
	notifier notification.Notifier,
) BroadcastHandler {
	return BroadcastHandler{
		logger:   appLogger,
		userRepo: userRepo,
		notifier: notifier,
	}
}

// Handle executes the broadcast logic.
func (h BroadcastHandler) Handle(ctx context.Context, cmd BroadcastCommand) (recipients int, err error) {
	userIDs, err := h.userRepo.FindAllIDs(ctx)
	if err != nil {
		return 0, err
	}

	successfulSends := 0
	for _, userID := range userIDs {
		// We ignore errors here to ensure the broadcast continues even if one user fails
		if notifyErr := h.notifier.Notify(userID, cmd.Message); notifyErr != nil {
			h.logger.Error("Failed to send broadcast to user", "userID", userID, "error", notifyErr)
		} else {
			successfulSends++
		}
		// Sleep briefly to avoid hitting Telegram's rate limits
		time.Sleep(100 * time.Millisecond)
	}
	return successfulSends, nil
}
