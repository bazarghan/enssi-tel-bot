package keyboards

import "gopkg.in/telebot.v4"

var (
	menu = &telebot.ReplyMarkup{ResizeKeyboard: true}
	// selector = &telebot.ReplyMarkup{}
)

func KeyboardMain() *telebot.ReplyMarkup {
	btnStartLearning := menu.Text("شروع یادگیری")
	btnChallenge := menu.Text("ادعا داری ؟ بیا تو")
	btnProfile := menu.Text("پروفایل")
	btnSettings := menu.Text("تنظیمات")

	menu.Reply(
		menu.Row(btnStartLearning),
		menu.Row(btnChallenge),
		menu.Row(btnProfile),
		menu.Row(btnSettings),
	)

	return menu
}
