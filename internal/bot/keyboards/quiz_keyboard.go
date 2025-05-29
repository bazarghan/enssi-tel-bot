package keyboards

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
)

// QuizQuestionOptions creates an inline keyboard for quiz question options.
// Each button callback data will be "quiz_ans:<attempt_id>:<question_id>:<option_id>"
func QuizQuestionOptions(options []models.QuizQuestionOption, attemptID uint, questionModelID uint) *telebot.ReplyMarkup {
	inlineMenu := &telebot.ReplyMarkup{}

	rows := make([]telebot.Row, len(options))
	for i, opt := range options {
		btn := inlineMenu.Data(
			opt.Text, // Button text
			fmt.Sprintf("quiz_ans:%d:%d:%d", attemptID, questionModelID, opt.ID), // Callback data
		)
		rows[i] = inlineMenu.Row(btn)
	}

	inlineMenu.Inline(rows...)
	return inlineMenu
}
