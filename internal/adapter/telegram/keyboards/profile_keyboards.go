package keyboards

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	ui "github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/uiconstants"
	"gopkg.in/telebot.v4"
)

// ProfileMenuKeyboard is shown when viewing the user profile.
func ProfileMenuKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	menu.Reply(
		menu.Row(menu.Text(ui.BtnViewAchievementsText)),
		menu.Row(menu.Text(ui.BtnReturnToMainMenuText)), // Assuming BtnReturnToMainMenu is defined elsewhere
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
		menu.Row(menu.Text(ui.BtnReturnToProfileText)),
		menu.Row(menu.Text(ui.BtnReturnToMainMenuText)),
	)

	menu.Reply(rows...)
	return menu
}
