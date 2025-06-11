package telegram

import (
	"context"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"gopkg.in/telebot.v4"
)

// Notifier is an adapter that implements the notification.Notifier port using Telebot.
type Notifier struct {
	bot      *telebot.Bot
	userRepo user.Repository
}

// NewNotifier creates a new Telegram notifier.
func NewNotifier(bot *telebot.Bot, userRepo user.Repository) *Notifier {
	return &Notifier{bot: bot, userRepo: userRepo}
}

// Notify sends a message to a user via Telegram.
func (n *Notifier) Notify(userID uint, message string) error {
	telegramID, err := n.userRepo.FindTelegramID(context.Background(), userID)
	if err != nil {
		return fmt.Errorf("could not find telegram ID for user %d: %w", userID, err)
	}

	if telegramID == 0 {
		return fmt.Errorf("user %d has a zero telegram ID, cannot notify", userID)
	}

	userRecipient := &telebot.User{ID: telegramID}

	_, err = n.bot.Send(userRecipient, message)
	return err
}
