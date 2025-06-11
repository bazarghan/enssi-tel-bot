package keyboards

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	"gopkg.in/telebot.v4"
)

var (
	BtnViewAchievements = telebot.Btn{Text: "🏆 دستاوردها"}
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

// AchievementsListKeyboard shows earned achievements as inline buttons.
func AchievementsListKeyboard(achievements []dto.AchievementView) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row
	for _, ach := range achievements {
		callbackData := fmt.Sprintf("%s%d", ShowAchievementCallbackPrefix, ach.ID) // Assumes const is defined
		btn := menu.Data(fmt.Sprintf("🏆 %s", ach.Title), callbackData)
		rows = append(rows, menu.Row(btn))
	}
	menu.Inline(rows...)
	return menu
}
