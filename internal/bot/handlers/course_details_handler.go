package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
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

	// 4) Upsert the UserCourse (start tracking progress)
	uc := models.UserCourse{UserID: user.ID, CourseID: course.ID}
	db.FirstOrCreate(&uc, uc) // ignore error for now

	// 5) Fetch the first 10 CourseWord entries for this course
	var cws []models.CourseWord
	if err := db.
		Where("course_id = ?", course.ID).
		Order("index asc").
		Limit(10).
		Find(&cws).Error; err != nil {
		return ctx.Send("خطا در بارگذاری کلمات دوره")
	}

	// 6) Lookup each Word title
	titles := make([]string, 0, len(cws))
	for _, cw := range cws {
		var w models.Word
		if err := db.First(&w, cw.WordID).Error; err == nil {
			titles = append(titles, w.Title)
		}
	}

	// 7) Send the first-10 list + course-details keyboard
	resp := fmt.Sprintf(
		"📖 دوره «%s» انتخاب شد. اولین ۱۰ کلمه:\n\n%s",
		course.PersianTitle,
		strings.Join(titles, "\n"),
	)
	return ctx.Send(
		resp,
		keyboards.Course(),
	)
}

