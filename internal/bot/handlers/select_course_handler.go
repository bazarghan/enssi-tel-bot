package handlers

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

func handleCourseSelection(ctx telebot.Context, db *gorm.DB) error {
	label := ctx.Text()

	if label == "بازگشت به منوی اصلی" {
		return handleReturnToMainMenu(ctx, db)
	}

	var course models.Course
	if err := db.Where("persian_title = ?", label).First(&course).Error; err != nil {
		return nil
	}

	sender := ctx.Sender()
	tgID := int64(sender.ID)

	var user models.User
	if err := db.Where("telegram_id = ?", tgID).First(&user).Error; err != nil {
		return err
	}

	user.LastMenu = fmt.Sprintf("courseId:%d", course.ID)
	if err := db.Save(&user).Error; err != nil {
		return err
	}

	desc := course.PersianDescription
	return ctx.Send(
		desc,
		keyboards.CourseDetails(),
	)
}
