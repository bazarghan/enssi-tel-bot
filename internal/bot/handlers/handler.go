package handlers

import (
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"gopkg.in/telebot.v4"
)

func RegisterHandlers(bot *telebot.Bot) {
	bot.Handle("/start", handleStart)
}

func handleStart(ctx telebot.Context) error {
	menu := keyboards.KeyboardMain()

	return ctx.Send("Hello!", menu)
}
