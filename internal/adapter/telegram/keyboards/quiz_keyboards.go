package keyboards

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"gopkg.in/telebot.v4"
)

const (
	QuizAnswerCallbackPrefix = "quiz_ans:"
)

// QuizQuestionOptionsKeyboard generates the inline keyboard for a quiz question.
func QuizQuestionOptionsKeyboard(options []quiz.Option, attemptID uint) *telebot.ReplyMarkup {
	inlineMenu := &telebot.ReplyMarkup{}

	rows := make([]telebot.Row, 0, (len(options)+1)/2)

	optionsText := []string{
		"گزینه یک",
		"گزینه دو",
		"گزینه سه",
		"گزینه چهار",
		"گزینه پنج",
		"گزینه شش",
	}

	for i := 0; i < len(options); i += 2 {

		opt1 := options[i]
		data1 := fmt.Sprintf("%s%d:%d", QuizAnswerCallbackPrefix, attemptID, opt1.ID)
		btn1 := inlineMenu.Data(optionsText[i], data1)

		if i+1 < len(options) {
			opt2 := options[i+1]
			data2 := fmt.Sprintf("%s%d:%d", QuizAnswerCallbackPrefix, attemptID, opt2.ID)
			btn2 := inlineMenu.Data(optionsText[i+1], data2)
			rows = append(rows, inlineMenu.Row(btn2, btn1))
		} else {
			rows = append(rows, inlineMenu.Row(btn1))
		}
	}

	inlineMenu.Inline(rows...)
	return inlineMenu
}
