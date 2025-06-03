package bot

import (
	"log"

	"github.com/2000ostd/enssi-tel-bot/internal/services"
	"gopkg.in/telebot.v4"
)

// UserContextKey prevents key collisions inside telebot.Context.
type UserContextKey string

const (
	DBUserKey UserContextKey = "dbUser"
)

// UserActivityMiddleware ensures a user row exists and stores it in the update context.
func UserActivityMiddleware(appServices *services.AppServices) telebot.MiddlewareFunc {
	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) error {
			// Channel posts (and many service updates) have no sender.
			sender := c.Sender()
			if sender == nil {
				return next(c)
			}

			user, err := appServices.User().GetOrCreateUserByTelegramID(
				sender.ID,
				sender.Username,
				sender.FirstName,
				sender.LastName,
			)
			if err != nil {
				log.Printf("[UserActivityMiddleware] FATAL: telegramID=%d: %v", sender.ID, err)
				_ = c.Send("I'm having trouble setting up your account right now. Please try again later.")
				return nil // stop here; we already informed the user
			}

			// Make the user available to all subsequent handlers in this update.
			c.Set(string(DBUserKey), user)

			return next(c)
		}
	}
}

// ErrorHandlerMiddleware provides one place to log + notify on handler errors.
func ErrorHandlerMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if err := next(c); err != nil {
			// Build a safe log line (all fields may be absent).
			var (
				userID int64
				chatID int64
				text   string
				cbData string
			)
			if s := c.Sender(); s != nil {
				userID = s.ID
			}
			if ch := c.Chat(); ch != nil {
				chatID = ch.ID
			}
			if m := c.Message(); m != nil {
				text = m.Text
			}
			if cb := c.Callback(); cb != nil {
				cbData = cb.Data
			}

			log.Printf("[Handler ERROR] user=%d chat=%d text='%s' callback='%s' err=%v",
				userID, chatID, text, cbData, err)

			// Notify the user once, in a way appropriate to the update type.
			if c.Callback() != nil {
				_ = c.Respond(&telebot.CallbackResponse{
					Text:      "متاسفانه مشکلی پیش آمده، لطفا دوباره تلاش کنید.",
					ShowAlert: true,
				})
			} else {
				if sendErr := c.Send("متاسفانه مشکلی پیش آمده، لطفا دوباره تلاش کنید."); sendErr != nil {
					log.Printf("[ErrorHandlerMiddleware] failed to send error message: %v", sendErr)
				}
			}
		}
		return nil // error handled
	}
}
