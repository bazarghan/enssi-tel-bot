package keyboards

import (
	"fmt"
	"github.com/bazarghan/enssi-tel-bot/internal/domain/quiz"
	"gopkg.in/telebot.v4"

	ui "github.com/bazarghan/enssi-tel-bot/internal/adapter/telegram/uiconstants"
)

// QuizQuestionOptionsKeyboard generates the inline keyboard for a quiz question.
func QuizQuestionOptionsKeyboard(options []quiz.Option, attemptID uint, isAdmin bool) *telebot.ReplyMarkup {
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
		data1 := fmt.Sprintf("%s%d:%d", ui.CallbackPrefixQuizAnswer, attemptID, opt1.ID)
		btn1 := inlineMenu.Data(optionsText[i], data1)

		if i+1 < len(options) {
			opt2 := options[i+1]
			data2 := fmt.Sprintf("%s%d:%d", ui.CallbackPrefixQuizAnswer, attemptID, opt2.ID)
			btn2 := inlineMenu.Data(optionsText[i+1], data2)
			rows = append(rows, inlineMenu.Row(btn2, btn1))
		} else {
			rows = append(rows, inlineMenu.Row(btn1))
		}
	}

	if isAdmin {

		var correctOptionID uint
		for _, opt := range options {
			if opt.IsCorrect {
				correctOptionID = opt.ID
				break
			}
		}

		skipData := fmt.Sprintf("%s%d:%d", ui.CallbackPrefixQuizAnswer, attemptID, correctOptionID)
		btnSkip := inlineMenu.Data("سوال بعدی(Admin)", skipData)
		rows = append(rows, inlineMenu.Row(btnSkip))
	}

	inlineMenu.Inline(rows...)
	return inlineMenu
}
