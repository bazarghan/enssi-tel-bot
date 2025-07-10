package keyboards

import (
	ui "github.com/bazarghan/enssi-tel-bot/internal/adapter/telegram/uiconstants"
	"gopkg.in/telebot.v4"
)

// NewMainMenu creates the main menu keyboard, conditionally showing admin buttons.
func NewMainMenu(isAdmin bool, hasPendingReview bool) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	rows := []telebot.Row{
		menu.Row(menu.Text(ui.BtnStartLearningText)),
		menu.Row(menu.Text(ui.BtnMyProfileText)),
	}
	// Conditionally add the daily review button as the top option
	if hasPendingReview {
		rows = append(rows, menu.Row(menu.Text(ui.BtnDailyReviewText)))
	}

	if isAdmin {
		rows = append(rows, menu.Row(menu.Text(ui.BtnAdminPanelText)))
	}

	menu.Reply(rows...)
	return menu
}
