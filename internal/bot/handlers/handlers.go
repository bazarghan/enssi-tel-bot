package handlers

import (
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

func RegisterHandlers(bot *telebot.Bot, db *gorm.DB) {

	bot.Handle("/start", withTimestamp(db, func(ctx telebot.Context) error {
		return handleStart(ctx, db)
	}))

	bot.Handle(&keyboards.BtnStartLearning, withTimestamp(db, func(ctx telebot.Context) error {
		return handleBtnStartLearningClicked(ctx, db)
	}))

	bot.Handle(&keyboards.BtnPrevMenuCourse, withTimestamp(db, func(ctx telebot.Context) error {
		return handleBtnStartLearningClicked(ctx, db)
	}))

	bot.Handle(&keyboards.BtnPrevMenuCourseDetails, withTimestamp(db, func(ctx telebot.Context) error {
		return handleBtnStartLearningClicked(ctx, db)
	}))

	bot.Handle(telebot.OnText, withTimestamp(db, func(ctx telebot.Context) error {
		return handleCourseSelection(ctx, db)
	}))

	bot.Handle(&keyboards.BtnReturnToMainMenu, withTimestamp(db, func(ctx telebot.Context) error {
		return handleBtnReturnToMainMenuClicked(ctx, db)
	}))

	bot.Handle(&keyboards.BtnStartCourse, withTimestamp(db, func(ctx telebot.Context) error {
		return handleBtnStartCourseClicked(ctx, db)
	}))

	bot.Handle(&keyboards.BtnNextWord, withTimestamp(db, func(ctx telebot.Context) error {
		return handleBtnNextWordClicked(ctx, db)
	}))

}

// withTimestamp returns a handler that first spawns a background update
// of last_active and last_online, then immediately invokes h.
func withTimestamp(db *gorm.DB, h func(ctx telebot.Context) error) func(ctx telebot.Context) error {

	return func(ctx telebot.Context) error {
		if sender := ctx.Sender(); sender != nil {
			tgID := int64(sender.ID)
			now := time.Now()

			go func(id int64, ts time.Time) {
				_ = db.Model(&models.User{}).
					Where("telegram_id = ?", id).
					Updates(map[string]interface{}{
						"last_active": ts,
						"last_online": ts,
					}).Error
			}(tgID, now)
		}
		return h(ctx)
	}
}

func handleStart(ctx telebot.Context, db *gorm.DB) error {

	sender := ctx.Sender()
	tgID := int64(sender.ID)

	user := models.User{TelegramID: tgID}
	if err := db.
		FirstOrCreate(&user, models.User{TelegramID: tgID, LastMenu: "main"}).
		Error; err != nil {
		return err
	}

	now := time.Now()
	user.LastActive = now
	user.LastOnline = now
	user.LastMenu = "main"
	if err := db.Save(&user).Error; err != nil {
		return err
	}
	menu := keyboards.Main()

	desc := Texts["start"]

	return ctx.Send(desc, menu, telebot.ModeMarkdownV2)
}

func handleBtnStartLearningClicked(ctx telebot.Context, db *gorm.DB) error {

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

func handleBtnReturnToMainMenuClicked(ctx telebot.Context, db *gorm.DB) error {

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
