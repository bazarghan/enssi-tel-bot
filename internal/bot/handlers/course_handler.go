package handlers

import (
	"gopkg.in/telebot.v4"
)

func handleBtnNextWordClicked(ctx telebot.Context) error {
	return ctx.Send("Next word")
}
