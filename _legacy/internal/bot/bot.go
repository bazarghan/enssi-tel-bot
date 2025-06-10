package bot

import (
	"log"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/services"
	"gopkg.in/telebot.v4"
)

// InitializeBot creates and configures the Telebot instance.
// It now also takes AppServices to pass to the router setup.
func InitializeBot(token string, appServices *services.AppServices) (*telebot.Bot, error) {
	if token == "" {
		log.Fatal("Telebot token is empty. Please check TEL_BOT_TOKEN environment variable.")
	}

	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		OnError: func(err error, c telebot.Context) {
			log.Printf("[Telebot ERROR] Error: %v", err)
			if c != nil {
				senderID := int64(0)
				if c.Sender() != nil {
					senderID = c.Sender().ID
				}
				chatID := int64(0)
				if c.Chat() != nil {
					chatID = c.Chat().ID
				}
				callbackData := "N/A"
				if c.Callback() != nil { // <--- ADD THIS CHECK
					callbackData = c.Callback().Data
				}
				log.Printf("[Telebot ERROR] Context: User %d, Chat %d, Text: %s, Data: %s",
					senderID, chatID, c.Text(), callbackData)
			}
		},
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	// Create the user lock manager instance.
	lockManager := NewUserLockManager()

	// Setup router and middleware, now passing the lock manager as well.
	RegisterRoutes(b, appServices, lockManager)

	return b, nil
}
