package keyboards

import (
	ui "github.com/bazarghan/enssi-tel-bot/internal/adapter/telegram/uiconstants"
	"gopkg.in/telebot.v4"
)

// InCourseNavigationKeyboard provides navigation within a course.
func InCourseNavigationKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	menu.Reply(
		menu.Row(menu.Text(ui.BtnNextWordText)),
		menu.Row(menu.Text(ui.BtnReturnToMainMenuText)),
	)
	return menu
}

// BackToCourseListKeyboard provides a simple keyboard to return to the course list.
func BackToCourseListKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	// In a future slice, this might have a "Review Course" button too.
	menu.Reply(
		menu.Row(menu.Text(ui.BtnReturnToMainMenuText)),
	)
	return menu
}
