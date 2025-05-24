package handlers

import (
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

func handleReturnToMainMenu(ctx telebot.Context, db *gorm.DB) error {

	sender := ctx.Sender()
	tgID := int64(sender.ID)

	var user models.User
	if err := db.Where("telegram_id = ?", tgID).First(&user).Error; err != nil {
		return err
	}

	user.LastMenu = "main"
	if err := db.Save(&user).Error; err != nil {
		return err
	}

	menu := keyboards.Main()

	desc := Texts["return_to_main_menu"]

	return ctx.Send(desc, menu)
}
