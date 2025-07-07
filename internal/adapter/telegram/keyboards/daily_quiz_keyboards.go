package keyboards

import (
	"gopkg.in/telebot.v4"

	ui "github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/uiconstants"
)

func DailyReviewMenuKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	menu.Reply(
		menu.Row(menu.Text(ui.BtnStartReviewQuizText)),
		menu.Row(menu.Text(ui.BtnReturnToMainMenuText)),
	)

	return menu
}
