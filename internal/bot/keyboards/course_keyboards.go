package keyboards

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/services/course" // For CourseSummaryView
	"gopkg.in/telebot.v4"
)

// CourseListKeyboard generates a reply keyboard listing available courses.
// Each course title button will be handled by a text handler.
func CourseListKeyboard(courses []course.CourseSummaryView) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true, OneTimeKeyboard: true} // OneTimeKeyboard might be good here

	rows := make([]telebot.Row, 0, len(courses)+1)
	for _, c := range courses {
		// Button text combines Title (and PersianTitle if significantly different or for clarity)
		// User selects based on this text.
		btnText := c.PersianTitle
		if c.Title != "" && c.Title != c.PersianTitle {
			btnText = fmt.Sprintf("%s (%s)", c.PersianTitle, c.Title)
		}
		rows = append(rows, menu.Row(menu.Text(btnText)))
	}
	rows = append(rows, menu.Row(BtnReturnToMainMenu)) // Shared button from keyboards.go
	menu.Reply(rows...)
	return menu
}

const (
	BtnStartCoursePrefix        = "شروع دوره:"      // For button text, payload contains ID
	BtnContinueCoursePrefix     = "ادامه دوره:"     // For button text, payload contains ID
	BtnViewCoursePrefix         = "مشاهده دوره:"    // Payload contains ID (alternative to full text match for selection)
	CourseDetailsCallbackPrefix = "course_details:" // course_details:<course_id>
	BtnNextWordAction           = "کلمه بعدی"
)

// CourseDetailsKeyboard generates reply keyboard for course overview.
// progressPercentage: <0 means not started, 0-100 for started/in progress.
func CourseDetailsKeyboard(courseID uint, progressPercentage int, isCompleted bool) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	// These are reply buttons, their text will be matched by text handlers.
	// The callback data pattern is more for inline buttons, but shown for conceptual clarity.
	// The actual handler for these text buttons will need to parse the courseID from user's LastMenu or context.
	// For simplicity with text handlers, we can just use generic text and derive courseID from state.

	// Let's make the buttons simpler for text matching and rely on user.LastMenu for courseID context.
	var simpleActionText string
	if isCompleted {
		simpleActionText = "مرور دوره" // Review Course
	} else if progressPercentage >= 0 {
		simpleActionText = fmt.Sprintf("ادامه دوره (%d%%)", progressPercentage) // Continue Course (X%)
	} else {
		simpleActionText = "شروع دوره" // Start Course
	}

	menu.Reply(
		menu.Row(menu.Text(simpleActionText)), // This text will be handled
		menu.Row(BtnReturnToMainMenu),         // Text for "Return to Course List" or "Return to Main Menu"
		// If you want "Return to Course List", define a new button and handler.
		// For now, BtnReturnToMainMenu will take them to the absolute main menu.
		// A "BackToCourseList" button would be: menu.Text("بازگشت به لیست دوره‌ها")
	)
	return menu
}

func InCourseNavigationKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	menu.Reply(
		menu.Row(menu.Text(BtnNextWordAction)),
		menu.Row(BtnReturnToMainMenu), // Or a more contextual "Exit Course" button
	)
	return menu
}

// CourseCompletedKeyboard shown when a course is finished.
func CourseCompletedKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	menu.Reply(
		menu.Row(BtnReturnToMainMenu), // Or "Back to Course List"
	)
	return menu
}

// Helper to generate callback data for selecting a course from an inline list
func CourseSelectionCallbackData(courseID uint) string {
	return fmt.Sprintf("%s%d", BtnViewCoursePrefix, courseID)
}

// InlineCourseListKeyboard (Optional, if you prefer inline for course selection)
func InlineCourseListKeyboard(courses []course.CourseSummaryView) *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{}
	rows := make([]telebot.Row, 0, len(courses))

	for _, c := range courses {
		btnText := c.PersianTitle
		if c.ProgressPercentage > 0 && !c.IsCompletedByUser {
			btnText = fmt.Sprintf("%s (%d%%)", btnText, c.ProgressPercentage)
		} else if c.IsCompletedByUser {
			btnText = fmt.Sprintf("%s (تکمیل شده)", btnText)
		}
		rows = append(rows, markup.Row(markup.Data(btnText, CourseSelectionCallbackData(c.ID))))
	}
	// markup.Inline(rows...) // This would create an inline keyboard
	// For now, sticking to ReplyKeyboards as per original structure.
	// If you want to switch, uncomment markup.Inline(rows...) and ensure handlers for BtnViewCoursePrefix exist.
	return markup // Placeholder if not used as inline.
}

// Constants for course-related button texts that might be matched directly.
const (
	StartCourseButtonText      = "شروع دوره"
	ContinueCourseButtonText   = "ادامه دوره" // Often with % appended
	ReviewCourseButtonText     = "مرور دوره"
	NextWordButtonText         = "کلمه بعدی"
	BackToCourseListButtonText = "بازگشت به لیست دوره‌ها"
)
