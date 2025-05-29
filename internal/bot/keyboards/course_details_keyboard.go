package keyboards

import (
	"fmt"
	"gopkg.in/telebot.v4"
)

// Static buttons can remain if their text is fixed.
var BtnPrevMenuCourseDetails = menu.Text("برگرد به منوی قبلی")

// BtnReturnToMainMenu is likely defined in keyboards.go and is fine.

// CourseDetails now accepts progressPercentage.
// progressPercentage:
//
//	-1: User hasn't started the course / no UserCourse record.
//	0-100: User has started, value is the percentage.
func CourseDetails(progressPercentage int) *telebot.ReplyMarkup {
	// It's good practice to create a new markup object each time to avoid shared state issues.
	dynamicMenu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	var startOrContinueButtonText string

	if progressPercentage >= 0 && progressPercentage <= 100 {
		// User has started or has a record (even at 0% if progress is 0 but UserCourse exists).
		// "ادامه دوره" (Continue Course) seems more appropriate if progress > 0.
		// If progress is 0 but UserCourse exists, "شروع دوره" might still be better.
		// Let's refine: if UserCourse exists and Progress > 0, show "Continue".
		// The -1 marker means "not started at all".
		// If progressPercentage is 0 (from (0/total)*100), it's effectively "شروع دوره" or "ادامه دوره - 0%".
		// Let's assume if progressPercentage >= 0, they have at least an entry.
		startOrContinueButtonText = fmt.Sprintf("ادامه دوره - %d%%", progressPercentage)
		if progressPercentage == 0 { // If progress is literally 0% (e.g., UserCourse.Progress is 0 or 1 on a long course)
			// You might prefer "شروع دوره" or "ادامه دوره - 0%"
			// For this implementation, "ادامه دوره - 0%" will be shown if progress is minimal.
			// If UserCourse.Progress is 0, the handler will likely send "شروع دوره".
			// This logic is based on the percentage passed.
		}
	} else {
		// User has not started (progressPercentage == -1).
		startOrContinueButtonText = "شروع دوره"
	}

	dynamicMenu.Reply(
		dynamicMenu.Row(dynamicMenu.Text(startOrContinueButtonText)), // Button with dynamic text
		dynamicMenu.Row(BtnReturnToMainMenu, BtnPrevMenuCourseDetails),
	)

	return dynamicMenu
}
