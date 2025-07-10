package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bazarghan/enssi-tel-bot/internal/adapter/telegram/keyboards"
	"github.com/bazarghan/enssi-tel-bot/internal/domain/quiz"
	"github.com/bazarghan/enssi-tel-bot/internal/domain/user"

	sc "github.com/bazarghan/enssi-tel-bot/internal/adapter/telegram/stateconstants"
	"github.com/bazarghan/enssi-tel-bot/internal/platform/observability/logger"
	"gopkg.in/telebot.v4"
)

// Notifier is an adapter that implements the notification.Notifier port using Telebot.
type Notifier struct {
	logger   logger.Logger
	bot      *telebot.Bot
	userRepo user.Repository
	quizRepo quiz.Repository
}

// NewNotifier creates a new Telegram notifier.
func NewNotifier(

	appLogger logger.Logger,
	bot *telebot.Bot,
	userRepo user.Repository,
	quizRepo quiz.Repository,

) *Notifier {
	return &Notifier{

		logger:   appLogger,
		bot:      bot,
		userRepo: userRepo,
		quizRepo: quizRepo,
	}
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

// SendMainMenu sends a message with the main menu keyboard attached.
// It dynamically includes the "Daily Review" button because we know one is pending.
func (n *Notifier) NotifyWithMainMenu(userID uint, message string) error {
	// 1. Find the user's Telegram ID
	telegramID, err := n.userRepo.FindTelegramID(context.Background(), userID)
	if err != nil {
		return fmt.Errorf("could not find telegramID for user %d: %w", userID, err)
	}

	// 2. Find the user's full details to check if they are an admin
	u, err := n.userRepo.FindByID(context.Background(), userID)
	if err != nil {
		return fmt.Errorf("could not find user %d to build main menu: %w", userID, err)
	}

	if strings.HasPrefix(u.LastMenu, "in_quiz:") {

		parts := strings.Split(u.LastMenu, ":")
		if len(parts) == 3 {

			quizType := parts[1]
			attemptID, err := strconv.ParseUint(parts[2], 10, 64)

			if err == nil {
				// 2. Fetch the quiz attempt from the database to get the message ID.
				attempt, err := n.quizRepo.GetAttempt(context.Background(), uint(attemptID))
				if err != nil {

					n.logger.Error("Could not get quiz attempt to edit message", "attempt_id", attemptID, "error", err)
					return err

				} else if attempt.CurrentQuestionMessageID != 0 {

					var msgText string

					if quizType == "review" {
						// If it's a review quiz, delete the attempt.
						n.quizRepo.DeleteAttempt(context.Background(), uint(attemptID))
						msgText = "آزمون مرور لغو شد. به منوی اصلی بازگشتید."
					} else {
						// For other quizzes (like course quizzes), just pause them.
						msgText = "آزمون متوقف شد. شما به منوی اصلی بازگشتید."
					}

					messageToEdit := &telebot.Message{
						ID:   attempt.CurrentQuestionMessageID,
						Chat: &telebot.Chat{ID: telegramID},
					}

					if _, err := n.bot.Edit(messageToEdit, msgText); err != nil {
						// This error is not critical, the user can still proceed.
						n.logger.Warn("Failed to edit old quiz message", "message_id", attempt.CurrentQuestionMessageID, "error", err)
					}
				}
			}
		}
	}

	if err := n.userRepo.UpdateLastMenu(context.Background(), u.ID, sc.StateMain); err != nil {
		n.logger.Error("Failed to update user menu to main for notification", "userID", userID, "error", err)
	}

	userRecipient := &telebot.User{ID: telegramID}

	// 3. Build the main menu, passing `true` for `hasPendingReview`
	mainMenuKeyboard := keyboards.NewMainMenu(u.IsAdmin, true)

	// 4. Send the message with the keyboard
	_, err = n.bot.Send(userRecipient, message, mainMenuKeyboard)
	return err

}
