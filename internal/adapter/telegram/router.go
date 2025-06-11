package telegram

import (
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/callback"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/command"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/message"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/di"
	registerCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/user"
	"gopkg.in/telebot.v4"
)

// RegisterRoutes sets up all command, text, and callback handlers for the bot.
func RegisterRoutes(
	b *telebot.Bot,
	appHandlers *di.BotApp,
	registerUserHandler registerCmd.RegisterUserHandler,
	lockManager *UserLockManager,
) {
	// Apply middleware in order: error handling, user activity, user lock.
	b.Use(ErrorHandlerMiddleware, UserActivityMiddleware(registerUserHandler), UserLockMiddleware(lockManager))

	// --- Command Handlers ---
	// The command handler instance is retrieved from the DI container.
	cmdHandler := appHandlers.CommandHandler
	b.Handle("/start", cmdHandler.HandleStart)
	b.Handle("/myprofile", cmdHandler.HandleMyProfile)

	// --- Message Handler ---
	// The message handler instance handles all non-command text messages.
	msgHandler := appHandlers.MessageHandler
	b.Handle(telebot.OnText, msgHandler.Handle)

	// --- Callback Handler ---
	// The callback handler instance handles all inline button presses.
	cbHandler := appHandlers.CallbackHandler
	b.Handle(telebot.OnCallback, cbHandler.Handle)
}
