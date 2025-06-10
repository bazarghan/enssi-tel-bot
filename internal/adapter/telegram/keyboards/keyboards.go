package keyboards

import "gopkg.in/telebot.v4"

// NewMainMenu creates the main menu keyboard, conditionally showing admin buttons.
func NewMainMenu(isAdmin bool) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	btnStartLearning := menu.Text("شروع یادگیری")
	btnMyProfile := menu.Text("پروفایل")
	btnAdminPanel := menu.Text("پنل ادمین")

	rows := []telebot.Row{
		menu.Row(btnStartLearning),
		menu.Row(btnMyProfile),
	}

	if isAdmin {
		rows = append(rows, menu.Row(btnAdminPanel))
	}

	menu.Reply(rows...)
	return menu
}
