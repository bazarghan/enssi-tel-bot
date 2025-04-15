package keyboards

import "gopkg.in/telebot.v4"

var (
	Btn504  = menu.Text("مجموعه 504 کلمه")
	Btn1100 = menu.Text("مجموعه 1100 کلمه")
)

func SelectCourse() *telebot.ReplyMarkup {
	menu.Reply(
		menu.Row(Btn504),
		menu.Row(Btn1100),
		menu.Row(BtnReturnToMainMenu),
	)

	return menu
}
