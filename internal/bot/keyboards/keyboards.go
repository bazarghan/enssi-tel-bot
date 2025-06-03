package keyboards

import "gopkg.in/telebot.v4"

var (
	// MainMenu is the primary reply keyboard markup.
	MainMenu = &telebot.ReplyMarkup{ResizeKeyboard: true}

	// Buttons for the main menu
	BtnStartLearning    = MainMenu.Text("شروع یادگیری")        // 📚 Start Learning
	BtnMyProfile        = MainMenu.Text("پروفایل")             // 👤 My Profile
	BtnSettings         = MainMenu.Text("تنظیمات")             // ⚙️ Settings
	BtnReturnToMainMenu = MainMenu.Text("بازگشت به منوی اصلی") // ↩️ Return to Main Menu
)

func init() {
	MainMenu.Reply(
		MainMenu.Row(BtnStartLearning),
		MainMenu.Row(BtnMyProfile /*, BtnSettings*/), // Settings can be added later
	)
}

// BackToMainMenuKeyboard provides a simple keyboard with a "Return to Main Menu" button.
func BackToMainMenuKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	menu.Reply(
		menu.Row(BtnReturnToMainMenu),
	)
	return menu
}
