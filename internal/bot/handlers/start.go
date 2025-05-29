package handlers

import (
	"strings"
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
		return handleStartLearning(ctx, db)
	}))

	bot.Handle(&keyboards.BtnPrevMenuCourse, withTimestamp(db, func(ctx telebot.Context) error {
		return handleStartLearning(ctx, db)
	}))

	bot.Handle(&keyboards.BtnPrevMenuCourseDetails, withTimestamp(db, func(ctx telebot.Context) error {
		return handleStartLearning(ctx, db)
	}))

	bot.Handle(telebot.OnText, withTimestamp(db, func(ctx telebot.Context) error {
		text := ctx.Text()
		if text == "شروع دوره" || strings.HasPrefix(text, "ادامه دوره - ") {
			return handleStartCourse(ctx, db)
		} else {
			return handleCourseSelection(ctx, db)
		}
	}))

	bot.Handle(&keyboards.BtnReturnToMainMenu, withTimestamp(db, func(ctx telebot.Context) error {
		return handleReturnToMainMenu(ctx, db)
	}))

	// bot.Handle(&keyboards.BtnStartCourse, withTimestamp(db, func(ctx telebot.Context) error {
	// 	return handleStartCourse(ctx, db)
	// }))
	//
	bot.Handle(&keyboards.BtnNextWord, withTimestamp(db, func(ctx telebot.Context) error {
		return handleNextWord(ctx, db)
	}))

	// --- Handler for Quiz Answer Callbacks ---
	bot.Handle(telebot.OnCallback, withTimestamp(db, func(ctx telebot.Context) error {
		if ctx.Callback() == nil {
			return nil
		}

		callbackData := strings.TrimSpace(ctx.Callback().Data)

		expectedPrefix := "quiz_ans:"
		if strings.HasPrefix(callbackData, expectedPrefix) {
			return handleQuizAnswerCallback(ctx, db)
		}
		return nil
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

	user, err := fetchUser(ctx, db)
	if err != nil {
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
