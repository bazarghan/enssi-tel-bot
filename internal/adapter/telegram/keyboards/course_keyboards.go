package keyboards

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	"gopkg.in/telebot.v4"
)

const (
	StartCourseButtonText    = "شروع دوره"
	ContinueCourseButtonText = "ادامه دوره"
	ReviewCourseButtonText   = "مرور دوره"
	BtnReturnToMainMenu      = "بازگشت به منوی اصلی"
)

// CourseListKeyboard generates a reply keyboard listing available courses.
func CourseListKeyboard(courses []dto.CourseSummary) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true, OneTimeKeyboard: true}

	rows := make([]telebot.Row, 0, len(courses)+1)
	for _, c := range courses {
		btnText := c.PersianTitle
		if c.ProgressPercentage > 0 && !c.IsCompleted {
			btnText = fmt.Sprintf("%s (%d%%)", btnText, c.ProgressPercentage)
		} else if c.IsCompleted {
			btnText = fmt.Sprintf("%s (تکمیل شده)", btnText)
		}
		rows = append(rows, menu.Row(menu.Text(btnText)))
	}
	rows = append(rows, menu.Row(menu.Text(BtnReturnToMainMenu)))
	menu.Reply(rows...)
	return menu
}

// CourseDetailsKeyboard generates reply keyboard for course overview.
func CourseDetailsKeyboard(co dto.CourseOverview) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	var actionButtonText string
	if co.IsCompleted {
		actionButtonText = ReviewCourseButtonText
	} else if co.IsStarted {
		actionButtonText = fmt.Sprintf("%s (%d%%)", ContinueCourseButtonText, co.ProgressPercentage)
	} else {
		actionButtonText = StartCourseButtonText
	}

	menu.Reply(
		menu.Row(menu.Text(actionButtonText)),
		menu.Row(menu.Text(BtnReturnToMainMenu)),
	)
	return menu
}

