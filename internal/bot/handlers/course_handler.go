package handlers

import (
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

func handleBtnNextWordClicked(ctx telebot.Context, db *gorm.DB) error {
	return ctx.Send("Next word")
}
