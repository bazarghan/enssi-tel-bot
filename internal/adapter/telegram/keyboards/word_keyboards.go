package keyboards

import "gopkg.in/telebot.v4"

const (
	NextWordButtonText = "کلمه بعدی"
)

// InCourseNavigationKeyboard provides navigation within a course.
func InCourseNavigationKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	menu.Reply(
		menu.Row(menu.Text(NextWordButtonText)),
		menu.Row(menu.Text(BtnReturnToMainMenu)),
	)
	return menu
}

// BackToCourseListKeyboard provides a simple keyboard to return to the course list.
func BackToCourseListKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	// In a future slice, this might have a "Review Course" button too.
	menu.Reply(
		menu.Row(menu.Text(BtnReturnToMainMenu)),
	)
	return menu
}
