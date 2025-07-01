package telegram

import (
	"context"

	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	registerCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/user"
	"gopkg.in/telebot.v4"
)

// UserContextKey is the key for storing the domain user object in the context.
const UserContextKey = "dbUser"

// UserActivityMiddleware ensures a user entity exists and stores it in the context.
// It uses the RegisterUserHandler use case, decoupling it from the persistence layer.
func UserActivityMiddleware(appLogger logger.Logger, registerUserHandler registerCmd.RegisterUserHandler) telebot.MiddlewareFunc {
	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) error {
			sender := c.Sender()
			if sender == nil {
				return next(c) // Not a user-initiated update
			}

			cmd := registerCmd.RegisterUserCommand{
				TelegramID: sender.ID,
				Username:   sender.Username,
				FirstName:  sender.FirstName,
				LastName:   sender.LastName,
			}

			// Execute the use case to get or create the user.
			result, err := registerUserHandler.Handle(context.Background(), cmd)
			if err != nil {
				appLogger.Error("Failed to register or find user in middleware", "error", err, "telegramID", sender.ID)
				_ = c.Send("I'm having trouble with your account right now. Please try again later.")
				return nil // Stop processing
			}

			// Store the pure domain user entity in the context for other handlers.
			c.Set(UserContextKey, result.User)

			return next(c)
		}
	}
}

// UserLockMiddleware prevents concurrent request processing for the same user.
func UserLockMiddleware(appLogger logger.Logger, lockManager *UserLockManager) telebot.MiddlewareFunc {
	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) error {
			sender := c.Sender()
			if sender == nil {
				return next(c)
			}
			userID := sender.ID

			if !lockManager.TryLock(userID) {
				appLogger.Warn("Ignored concurrent request for user", "userID", userID)
				if cb := c.Callback(); cb != nil {
					c.Respond() // Acknowledge callback to stop loading animation
				}
				return nil // Stop processing
			}
			defer lockManager.Unlock(userID)

			return next(c)
		}
	}
}

// ErrorHandlerMiddleware logs handler errors and notifies the user.
func ErrorHandlerMiddleware(appLogger logger.Logger) telebot.MiddlewareFunc {
	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) error {
			if err := next(c); err != nil {
				appLogger.Error(
					"Handler error",
					"error", err,
					"senderID", c.Sender().ID,
				)
				if c.Callback() != nil {
					_ = c.Respond(&telebot.CallbackResponse{
						Text:      "An unexpected error occurred.",
						ShowAlert: true,
					})
				} else {
					_ = c.Send("An unexpected error occurred. Please try again.")
				}
			}
			return nil // Error is handled
		}
	}
}
