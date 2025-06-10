package bot

import (
	"context"
	"log"

	registerCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/user"
	"gopkg.in/telebot.v4"
)

// UserContextKey prevents key collisions inside telebot.Context.
type UserContextKey string

const (
	DBUserKey UserContextKey = "dbUser"
)

// UserActivityMiddleware ensures a user row exists and stores it in the update context.
// func UserActivityMiddleware(appServices *services.AppServices) telebot.MiddlewareFunc {

func UserActivityMiddleware(registerUserHandler registerCmd.RegisterUserHandler) telebot.MiddlewareFunc {

	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) error {
			// Channel posts (and many service updates) have no sender.
			sender := c.Sender()
			if sender == nil {
				return next(c)
			}

			cmd := registerCmd.RegisterUserCommand{
				TelegramID: sender.ID,
				Username:   sender.Username,
				FirstName:  sender.FirstName,
				LastName:   sender.LastName,
			}

			// We run the command here to ensure the user exists for every interaction.
			// The result contains the domain user entity.
			result, err := registerUserHandler.Handle(context.Background(), cmd)

			if err != nil {
				log.Printf("[UserActivityMiddleware] FATAL: telegramID=%d: %v", sender.ID, err)
				_ = c.Send("I'm having trouble setting up your account right now. Please try again later.")
				return nil // stop here; we already informed the user
			}
			c.Set(string(DBUserKey), result.User)

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

// UserLockMiddleware prevents concurrent request processing for the same user.
// It should be used in the router chain after UserActivityMiddleware.
func UserLockMiddleware(lockManager *UserLockManager) telebot.MiddlewareFunc {
	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) error {
			sender := c.Sender()
			if sender == nil {
				// Not a user-initiated update (e.g., channel post), so we don't lock.
				return next(c)
			}
			userID := sender.ID

			// Try to acquire the lock for this user.
			if !lockManager.TryLock(userID) {
				// User is already locked, meaning another request from them is still being processed.
				// We log it and ignore this new request by returning nil.
				log.Printf("[UserLockMiddleware] Ignored concurrent request for UserID %d. User is locked.", userID)

				// If this was a callback query (from an inline button), we should still "respond" to it
				// to stop the loading animation on the user's client, even though we're ignoring it.
				if cb := c.Callback(); cb != nil {
					// Respond without any text. This just acknowledges the button press.
					c.Respond()
				}

				return nil // Stop processing this duplicate/concurrent request.
			}

			// If we got here, the lock was acquired successfully.
			// We use `defer` to GUARANTEE that the lock is released when this handler function
			// (and all the handlers after it in the chain) finishes, no matter what.
			defer lockManager.Unlock(userID)

			// Proceed to the next handler in the chain (e.g., the actual command/message handler).
			return next(c)
		}
	}
}

// ---> END OF NEW MIDDLEWARE FUNCTION <---
