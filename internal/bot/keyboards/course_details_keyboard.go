package keyboards

import (
	"fmt"
	"gopkg.in/telebot.v4"
)

var BtnPrevMenuCourseDetails = menu.Text("برگرد به منوی قبلی")

func CourseDetails(progressPercentage int) *telebot.ReplyMarkup {
	dynamicMenu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	var startOrContinueButtonText string

	if progressPercentage >= 0 && progressPercentage <= 100 {

		startOrContinueButtonText = fmt.Sprintf("ادامه دوره - %d%%", progressPercentage)

	} else {
		startOrContinueButtonText = "شروع دوره"
	}

	dynamicMenu.Reply(
		dynamicMenu.Row(dynamicMenu.Text(startOrContinueButtonText)),
		dynamicMenu.Row(BtnReturnToMainMenu, BtnPrevMenuCourseDetails),
	)

	return dynamicMenu
}
