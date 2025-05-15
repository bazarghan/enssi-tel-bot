package keyboards

import (
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
)

func CourseMenu(courses []models.Course) *telebot.ReplyMarkup {
	mk := &telebot.ReplyMarkup{
		ResizeKeyboard:  true,
		OneTimeKeyboard: true,
	}

	rows := make([]telebot.Row, 0, len(courses)+1)
	for _, c := range courses {
		btn := mk.Text(c.PersianTitle)
		rows = append(rows, mk.Row(btn))
	}
	back := mk.Text("بازگشت به منوی اصلی")
	rows = append(rows, mk.Row(back))

	mk.Reply(rows...)
	return mk
}
