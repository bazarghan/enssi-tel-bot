package telegram

import (
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/callback"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/command"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/message"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	registerCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/user"
	"gopkg.in/telebot.v4"
)

// RegisterRoutes sets up all command, text, and callback handlers for the bot.
func RegisterRoutes(
	b *telebot.Bot,
	appLogger logger.Logger,
	cmdHandler *command.Handler,
	msgHandler *message.Handler,
	cbHandler *callback.Handler,
	registerUserHandler registerCmd.RegisterUserHandler,
	lockManager *UserLockManager,
) {

	// Apply middleware in order: error handling, user activity, user lock.
	b.Use(
		ErrorHandlerMiddleware(appLogger),
		UserActivityMiddleware(appLogger, registerUserHandler),
		UserLockMiddleware(appLogger, lockManager),
	)

	// --- Command Handlers ---
	b.Handle("/start", cmdHandler.HandleStart)
	b.Handle("/myprofile", cmdHandler.HandleMyProfile)

	// --- Message Handler ---
	b.Handle(telebot.OnText, msgHandler.Handle)

	// --- Callback Handler ---
	b.Handle(telebot.OnCallback, cbHandler.Handle)
}
