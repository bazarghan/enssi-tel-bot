package admin

import (
	"context"
	"log"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/domain/notification"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
)

type BroadcastCommand struct {
	Message string
}

type BroadcastHandler struct {
	logger   logger.Logger
	userRepo user.Repository
	notifier notification.Notifier
}

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

func (h BroadcastHandler) Handle(ctx context.Context, cmd BroadcastCommand) (recipients int, err error) {
	userIDs, err := h.userRepo.FindAllIDs(ctx)
	if err != nil {
		return 0, err
	}

	for _, userID := range userIDs {
		// We ignore errors here to ensure the broadcast continues even if one user fails
		if notifyErr := h.notifier.Notify(userID, cmd.Message); notifyErr != nil {
			log.Printf("Failed to send broadcast to user %d: %v", userID, notifyErr)
		}
		// Sleep briefly to avoid hitting Telegram's rate limits
		time.Sleep(100 * time.Millisecond)
	}
	return len(userIDs), nil
}
