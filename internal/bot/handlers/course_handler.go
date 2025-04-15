package handlers

import (
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"gopkg.in/telebot.v4"
)

func handleBtnNextWordClicked(ctx telebot.Context) error {
	return ctx.Send("Next word")
}

func handleBtnPrevMenuCourseClicked(ctx telebot.Context) error {
	menu := keyboards.SelectCourse()

	return ctx.Send("Select a course or continue one", menu)
}
