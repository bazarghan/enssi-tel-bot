package keyboards

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/models" // For models.QuizQuestionOption
	"gopkg.in/telebot.v4"
)

const (
	QuizAnswerCallbackPrefix = "quiz_ans:" // quiz_ans:<attempt_ID>:<option_ID>
)

// QuizQuestionOptionsKeyboard creates an inline keyboard for quiz question options.
func QuizQuestionOptionsKeyboard(options []models.QuizQuestionOption, attemptID uint) *telebot.ReplyMarkup {
	inlineMenu := &telebot.ReplyMarkup{}

	rows := make([]telebot.Row, len(options))
	for i, opt := range options {
		// Callback data: "quiz_ans:<attempt_id>:<option_id>"
		// Question ID is not strictly needed in the callback if the service can derive it from attemptID + optionID,
		// or if the service's SubmitAnswer only needs attemptID and chosenOptionID.
		// The service `SubmitAnswer(attemptID uint, chosenOptionID uint, userID uint)` confirms this.
		callbackData := fmt.Sprintf("%s%d:%d", QuizAnswerCallbackPrefix, attemptID, opt.ID)
		btn := inlineMenu.Data(opt.Text, callbackData)
		rows[i] = inlineMenu.Row(btn)
	}

	inlineMenu.Inline(rows...)
	return inlineMenu
}
