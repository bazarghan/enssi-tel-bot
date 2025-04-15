package keyboards

import "gopkg.in/telebot.v4"

var (
	BtnNextWord       = menu.Text("کلمه بعدی")
	BtnPrevMenuCourse = menu.Text("برگرد به منوی قبلی")
)

func Course() *telebot.ReplyMarkup {
	menu.Reply(
		menu.Row(BtnNextWord),
		menu.Row(BtnReturnToMainMenu, BtnPrevMenuCourse),
	)

	return menu
}
