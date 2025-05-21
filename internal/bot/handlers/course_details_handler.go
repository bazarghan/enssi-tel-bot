package handlers

import (
	"errors"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

func handleBtnStartCourseClicked(ctx telebot.Context, db *gorm.DB) error {
	// 1) Load the user from DB
	tgID := ctx.Sender().ID
	var user models.User
	if err := db.
		Where("telegram_id = ?", tgID).
		First(&user).Error; err != nil {
		return ctx.Send("مشکلی در یافتن کاربر پیش آمد")
	}

	// 2) Parse courseId out of LastMenu
	const prefix = "courseId:"
	if !strings.HasPrefix(user.LastMenu, prefix) {
		return ctx.Send("ابتدا باید یک دوره انتخاب کنید")
	}
	idStr := strings.TrimPrefix(user.LastMenu, prefix)
	cid, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return ctx.Send("خطا در خواندن شناسه دوره")
	}
	courseID := uint(cid)

	// 3) Load the Course
	var course models.Course
	if err := db.First(&course, courseID).Error; err != nil {
		return ctx.Send("دوره‌ای با این شناسه پیدا نشد")
	}

	var uc models.UserCourse
	if err := db.
		Where("user_id = ? AND course_id = ?", user.ID, course.ID).
		First(&uc).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// not enrolled yet → start at 1
			uc = models.UserCourse{
				UserID:   user.ID,
				CourseID: course.ID,
				Progress: 1,
			}
			if err := db.Create(&uc).Error; err != nil {
				return ctx.Send("خطا در ثبت پیشرفت دوره")
			}
		} else {
			return ctx.Send("خطا در بررسی دوره کاربر")
		}
	}

	if err := sendCourseWord(ctx, db, tgID, course.ID, uc.Progress); err != nil {
		return err
	}

	return ctx.Send("برای دریافت کلمه بعدی رو کلمه بعدی کلیک کنید", keyboards.Course())

}
