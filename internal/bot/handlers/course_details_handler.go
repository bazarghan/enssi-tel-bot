package handlers

import (
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"gopkg.in/telebot.v4"
)

func handleBtnStartCourseClicked(ctx telebot.Context) error {
	menu := keyboards.Course()

	return ctx.Send("Course", menu)
}

func handleBtnPrevMenuCourseDetailsClicked(ctx telebot.Context) error {
	menu := keyboards.SelectCourse()

	return ctx.Send("Select a course or continue one", menu)
}
