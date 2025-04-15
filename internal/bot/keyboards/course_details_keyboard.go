package keyboards

import "gopkg.in/telebot.v4"

var (
	BtnStartCourse           = menu.Text("شروع دوره")
	BtnPrevMenuCourseDetails = menu.Text("برگرد به منوی قبلی")
)

func CourseDetails() *telebot.ReplyMarkup {
	menu.Reply(
		menu.Row(BtnStartCourse),
		menu.Row(BtnReturnToMainMenu, BtnPrevMenuCourseDetails),
	)

	return menu
}
