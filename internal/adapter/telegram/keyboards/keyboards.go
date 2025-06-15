package keyboards

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	"gopkg.in/telebot.v4"
)

var (
	BtnViewAchievements = telebot.Btn{Text: "🏆 دستاوردها"}
	BtnReturnToProfile  = telebot.Btn{Text: "بازگشت به پروفایل"}
)

const ShowAchievementCallbackPrefix = "ach_show:"

// NewMainMenu creates the main menu keyboard, conditionally showing admin buttons.
func NewMainMenu(isAdmin bool) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	btnStartLearning := menu.Text("شروع یادگیری")
	btnMyProfile := menu.Text("پروفایل")
	btnAdminPanel := menu.Text("پنل ادمین")

	rows := []telebot.Row{
		menu.Row(btnStartLearning),
		menu.Row(btnMyProfile),
	}

	if isAdmin {
		rows = append(rows, menu.Row(btnAdminPanel))
	}

	menu.Reply(rows...)
	return menu
}

// ProfileMenuKeyboard is shown when viewing the user profile.
func ProfileMenuKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	menu.Reply(
		menu.Row(BtnViewAchievements),
		menu.Row(menu.Text(BtnReturnToMainMenu)), // Assuming BtnReturnToMainMenu is defined elsewhere
	)
	return menu
}

func AchievementsListKeyboard(achievements []dto.AchievementView) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	var rows []telebot.Row
	var cols []telebot.Btn

	for _, ach := range achievements {
		btn := menu.Text(fmt.Sprintf("🏆 %s", ach.Title))
		cols = append(cols, btn)

		if len(cols) == 2 {
			rows = append(rows, menu.Row(cols...))
			cols = nil
		}
	}

	if len(cols) > 0 {
		rows = append(rows, menu.Row(cols...))
	}

	rows = append(
		rows,
		menu.Row(BtnReturnToProfile),
		menu.Row(menu.Text(BtnReturnToMainMenu)),
	)

	menu.Reply(rows...)
	return menu
}
