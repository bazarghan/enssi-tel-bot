package handlers

import (
	"errors"
	"log"
	"strconv"

	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

func handleReturnToMainMenu(ctx telebot.Context, db *gorm.DB) error {
	user, err := fetchUser(ctx, db)
	if err != nil {
		log.Printf("[ERROR] handleReturnToMainMenu: Error fetching user: %v", err)
		return ctx.Send("مشکلی در انجام درخواست شما پیش آمد.")
	}

	// --- New Logic: Check for active quiz and delete its current message ---
	activeAttempt, errFind := findAnyActiveQuizAttempt(db, user.ID) // This function was in course_handler.go

	if errFind != nil && !errors.Is(errFind, gorm.ErrRecordNotFound) {
		// Log actual DB error, but still allow user to go to main menu
		log.Printf("[ERROR] handleReturnToMainMenu: Error finding active quiz for UserID %d: %v", user.ID, errFind)
	}

	if activeAttempt != nil && activeAttempt.CurrentQuestionMessageID != 0 {
		// An active quiz attempt exists and it has a displayed message ID.
		if ctx.Chat() != nil { // Ensure we have a chat context to delete from
			log.Printf("[INFO] handleReturnToMainMenu: UserID %d returning to main menu. Active QuizAttemptID: %d, CurrentQuestionMessageID: %d. Deleting message.",
				user.ID, activeAttempt.ID, activeAttempt.CurrentQuestionMessageID)

			messageToDelete := telebot.StoredMessage{
				MessageID: strconv.Itoa(activeAttempt.CurrentQuestionMessageID),
				ChatID:    ctx.Chat().ID,
			}

			// Attempt to delete the message
			if errDel := ctx.Bot().Delete(messageToDelete); errDel != nil {
				log.Printf("[WARN] handleReturnToMainMenu: Failed to delete quiz message (MsgID: %s, ChatID: %d) for QuizAttemptID %d: %v",
					messageToDelete.MessageID, messageToDelete.ChatID, activeAttempt.ID, errDel)
				// Continue even if deletion fails, but log it. The message might remain.
			} else {
				log.Printf("[INFO] handleReturnToMainMenu: Successfully deleted quiz message (MsgID: %s) for QuizAttemptID %d.",
					messageToDelete.MessageID, activeAttempt.ID)
			}

			// Reset CurrentQuestionMessageID in the database for this attempt,
			// so next time sendCurrentQuizQuestion will send a new message.
			if errUpdate := db.Model(&models.QuizAttempt{}).Where("id = ?", activeAttempt.ID).Update("current_question_message_id", 0).Error; errUpdate != nil {
				log.Printf("[ERROR] handleReturnToMainMenu: Failed to reset CurrentQuestionMessageID for QuizAttemptID %d: %v", activeAttempt.ID, errUpdate)
			}
		} else {
			log.Printf("[WARN] handleReturnToMainMenu: ctx.Chat() is nil for UserID %d. Cannot delete quiz message.", user.ID)
		}
		// The quiz attempt itself (IsCompleted=false, CurrentQuestionNum) remains, allowing resumption.
	}
	// --- End of new logic ---

	user.LastMenu = "main"
	if err := db.Save(user).Error; err != nil {
		log.Printf("[ERROR] handleReturnToMainMenu: Error saving user LastMenu: %v", err)
		return ctx.Send("خطایی در به‌روزرسانی وضعیت شما رخ داد.")
	}

	desc := Texts["return_to_main_menu"]
	if desc == "" { // Fallback if text isn't loaded
		desc = "شما به منوی اصلی بازگشتید."
	}

	return ctx.Send(desc, keyboards.Main())
}
