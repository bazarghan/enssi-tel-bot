package keyboards

import (
	"gopkg.in/telebot.v4"

	ui "github.com/bazarghan/enssi-tel-bot/internal/adapter/telegram/uiconstants"
)

// Add this new function to the file
func AdminPanelKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	menu.Reply(
		menu.Row(menu.Text(ui.BtnAdminStatsText), menu.Text(ui.BtnAdminBroadcastText)),
		menu.Row(menu.Text(ui.BtnReturnToMainMenuText)),
	)

	return menu
}

func QuizKeyboard() *telebot.ReplyMarkup {

	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	menu.Reply(
		menu.Row(menu.Text(ui.BtnReturnToMainMenuText)),
	)
	return menu

}
