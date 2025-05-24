package handlers

import (
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

func handleStartLearning(ctx telebot.Context, db *gorm.DB) error {

	sender := ctx.Sender()
	tgID := int64(sender.ID)

	var user models.User
	if err := db.Where("telegram_id = ?", tgID).First(&user).Error; err != nil {
		return err
	}

	user.LastMenu = "courses"
	if err := db.Save(&user).Error; err != nil {
		return err
	}

	var courses []models.Course
	if err := db.Find(&courses).Error; err != nil {
		return ctx.Send("مشکلی در بارگذاری دوره‌ها پیش آمد")
	}
	return ctx.Send(
		"کدوم دوره رو می‌خوای شروع کنی؟",
		keyboards.CourseMenu(courses),
	)
}

// func handleProfile() error {
// 	return
// }

// func handleSetting() error {
//
// }
