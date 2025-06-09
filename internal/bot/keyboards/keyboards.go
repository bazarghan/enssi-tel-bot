package keyboards

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/services/user"
	"gopkg.in/telebot.v4"
)

var (
	BtnStartLearning    = telebot.Btn{Text: "شروع یادگیری"}        // 📚 Start Learning
	BtnMyProfile        = telebot.Btn{Text: "پروفایل"}             // 👤 My Profile
	BtnSettings         = telebot.Btn{Text: "تنظیمات"}             // ⚙️ Settings (Example, if you add it)
	BtnReturnToMainMenu = telebot.Btn{Text: "بازگشت به منوی اصلی"} // ↩️ Return to Main Menu
	BtnAdminPanel       = telebot.Btn{Text: "پنل ادمین"}           // 🛡️ Admin Panel (New)

	BtnViewAchievements = telebot.Btn{Text: "دستاورد ها"} // 🏆 Achievements
)

const (
	ShowAchievementCallbackPrefix = "ach_show:"
)

func NewMainMenu(isAdmin bool) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	// Define the rows using the button variables
	rows := []telebot.Row{
		menu.Row(BtnStartLearning),
		menu.Row(BtnMyProfile /*, menu.Row(BtnSettings)*/), // Example if settings is added
	}

	if isAdmin {
		rows = append(rows, menu.Row(BtnAdminPanel))
	}

	menu.Reply(rows...)
	return menu
}

var MainMenu = NewMainMenu(false)

func ProfileMenuKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	menu.Reply(
		menu.Row(BtnViewAchievements),
		menu.Row(BtnReturnToMainMenu),
	)
	return menu
}

func AchievementsListKeyboard(achievements []user.AchievementView) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{}

	var rows []telebot.Row
	for _, ach := range achievements {
		// Callback data will be "ach_show:<achievement_id>"
		callbackData := fmt.Sprintf("%s%d", ShowAchievementCallbackPrefix, ach.ID)

		btn := menu.Data(fmt.Sprintf("🏆 %s", ach.Title), callbackData)
		rows = append(rows, menu.Row(btn))
	}

	menu.Inline(rows...)
	return menu
}

// BackToMainMenuKeyboard provides a simple keyboard with a "Return to Main Menu" button.
func BackToMainMenuKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	menu.Reply(
		menu.Row(BtnReturnToMainMenu),
	)
	return menu
}
