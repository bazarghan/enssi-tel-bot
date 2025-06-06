package keyboards

import "gopkg.in/telebot.v4"

var (
	BtnStartLearning    = telebot.Btn{Text: "شروع یادگیری"}        // 📚 Start Learning
	BtnMyProfile        = telebot.Btn{Text: "پروفایل"}             // 👤 My Profile
	BtnSettings         = telebot.Btn{Text: "تنظیمات"}             // ⚙️ Settings (Example, if you add it)
	BtnReturnToMainMenu = telebot.Btn{Text: "بازگشت به منوی اصلی"} // ↩️ Return to Main Menu
	BtnAdminPanel       = telebot.Btn{Text: "پنل ادمین"}           // 🛡️ Admin Panel (New)
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

// BackToMainMenuKeyboard provides a simple keyboard with a "Return to Main Menu" button.
func BackToMainMenuKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	menu.Reply(
		menu.Row(BtnReturnToMainMenu),
	)
	return menu
}
