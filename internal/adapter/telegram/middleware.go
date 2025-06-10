// internal/adapter/telegram/middleware.go
package telegram

import (
	"context"
	register_user_uc "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/user"
	"gopkg.in/telebot.v4"
)

// UserContextKey is the key for storing the user domain entity in the context.
const UserContextKey = "user"

// UserActivityMiddleware creates or updates a user and adds them to the context.
func UserActivityMiddleware(handler *register_user_uc.Handler) telebot.MiddlewareFunc {
	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) error {
			sender := c.Sender()
			if sender == nil {
				return next(c)
			}

			cmd := register_user_uc.Command{
				TelegramID: sender.ID,
				Username:   sender.Username,
				FirstName:  sender.FirstName,
				LastName:   sender.LastName,
			}

			// The context here is a background context for the operation
			user, err := handler.Handle(context.Background(), cmd)
			if err != nil {
				// Handle error appropriately
				c.Send("Sorry, there was a problem setting up your account.")
				return err
			}

			// Make the domain user object available to subsequent handlers
			c.Set(UserContextKey, user)

			return next(c)
		}
	}
}
