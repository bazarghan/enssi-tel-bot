package keyboards

import "gopkg.in/telebot.v4"

var (
	menu = &telebot.ReplyMarkup{ResizeKeyboard: true}
	// selector = &telebot.ReplyMarkup{}

	BtnReturnToMainMenu = menu.Text("برگرد به منوی اصلی")

	BtnStartLearning = menu.Text("شروع یادگیری")
	BtnProfile       = menu.Text("پروفایل")
	BtnSettings      = menu.Text("تنظیمات")
)

func Main() *telebot.ReplyMarkup {
	menu.Reply(
		menu.Row(BtnStartLearning),
		menu.Row(BtnProfile),
		menu.Row(BtnSettings),
	)

	return menu
}
