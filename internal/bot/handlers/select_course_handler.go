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
		return ctx.Send(
			"به منوی اصلی بازگشتید",
			keyboards.Main(),
		)
	}

	var course models.Course
	if err := db.Where("persian_title = ?", label).First(&course).Error; err != nil {
		return nil
	}
	//
	// user := models.User{}
	// db.Where("telegram_id = ?", ctx.Sender().ID).First(&user)
	// db.Create(&models.UserCourse{
	// 	UserID:   user.ID,
	// 	CourseID: course.ID,
	// })
	//
	// And send a confirmation or the first lesson
	return ctx.Send(
		fmt.Sprintf("شما دوره «%s» را انتخاب کردید! بیا شروع کنیم.", course.PersianTitle),
		keyboards.CourseDetails(),
	)
}
