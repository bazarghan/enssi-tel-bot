package keyboards

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"gopkg.in/telebot.v4"
	"math/rand"
)

// QuizQuestionOptionsKeyboard generates the inline keyboard for a quiz question.
func QuizQuestionOptionsKeyboard(options []quiz.Option, attemptID uint) *telebot.ReplyMarkup {
	inlineMenu := &telebot.ReplyMarkup{}

	// Shuffle options for display
	rand.Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})

	rows := make([]telebot.Row, len(options))
	for i, opt := range options {
		data := fmt.Sprintf("quiz_ans:%d:%d", attemptID, opt.ID)
		rows[i] = inlineMenu.Row(inlineMenu.Data(opt.Text, data))
	}

	inlineMenu.Inline(rows...)
	return inlineMenu
}
