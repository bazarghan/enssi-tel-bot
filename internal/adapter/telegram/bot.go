package telegram

import (
	"log"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/platform/di"
	registerCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/user"
	"gopkg.in/telebot.v4"
)

// InitializeBot creates and configures the Telebot instance.
func InitializeBot(
	token string,
	appHandlers *di.BotApp,
	registerUserHandler registerCmd.RegisterUserHandler,
) (*telebot.Bot, error) {
	if token == "" {
		log.Fatal("Telebot token is empty. Please check environment variables.")
	}

	pref := telebot.Settings{
		Token:   token,
		Poller:  &telebot.LongPoller{Timeout: 10 * time.Second},
		OnError: logAndFormatError,
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	// Create the user lock manager.
	lockManager := NewUserLockManager()

	// Setup router and middleware.
	RegisterRoutes(b, appHandlers, registerUserHandler, lockManager)

	return b, nil
}

// logAndFormatError is a detailed error logger for the telebot settings.
func logAndFormatError(err error, c telebot.Context) {
	log.Printf("[Telebot ERROR] Error: %v", err)
	if c == nil {
		return
	}
	senderID := int64(0)
	if c.Sender() != nil {
		senderID = c.Sender().ID
	}
	chatID := int64(0)
	if c.Chat() != nil {
		chatID = c.Chat().ID
	}
	callbackData := "N/A"
	if c.Callback() != nil {
		callbackData = c.Callback().Data
	}
	log.Printf("[Telebot ERROR] Context: User %d, Chat %d, Text: %s, Data: %s",
		senderID, chatID, c.Text(), callbackData)
}
