package telegram

import (
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/callback"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/command"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/message"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/config"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	registerCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/user"
	"gopkg.in/telebot.v4"
)

// InitializeBot creates and configures the Telebot instance.
func InitializeBot(
	cfg *config.Config,
	appLogger logger.Logger,
	token string,
	cmdHandler *command.Handler,
	msgHandler *message.Router,
	cbHandler *callback.Handler,
	registerUserHandler registerCmd.RegisterUserHandler,
) (*telebot.Bot, error) {

	if token == "" {
		appLogger.Error("Telebot token is empty. Please check environment variables.")
	}

	// --- REFACTORED: Use a closure to capture the logger ---
	onError := func(err error, c telebot.Context) {
		logAndFormatError(err, c, appLogger)
	}

	pref := telebot.Settings{
		Token:   token,
		Poller:  &telebot.LongPoller{Timeout: 10 * time.Second},
		OnError: onError,
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	// Create the user lock manager.
	lockManager := NewUserLockManager()

	// Setup router and middleware.
	RegisterRoutes(b, appLogger, cmdHandler, msgHandler, cbHandler, registerUserHandler, lockManager)

	return b, nil
}

// logAndFormatError is a detailed error logger for the telebot settings.
func logAndFormatError(err error, c telebot.Context, appLogger logger.Logger) {
	baseLogger := appLogger.With("system", "telebot")

	if c == nil {
		baseLogger.Error(err.Error())
		return
	}

	var senderID int64
	if c.Sender() != nil {
		senderID = c.Sender().ID
	}

	var chatID int64
	if c.Chat() != nil {
		chatID = c.Chat().ID
	}

	var callbackData string
	if c.Callback() != nil {
		callbackData = c.Callback().Data
	}

	baseLogger.Error(
		"Telebot context error",
		"error", err,
		"senderID", senderID,
		"chatID", chatID,
		"text", c.Text(),
		"callback", callbackData,
	)
}
