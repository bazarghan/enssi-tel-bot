package keyboards

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/services/course" // For CourseSummaryView
	"gopkg.in/telebot.v4"
)

// CourseListKeyboard generates a reply keyboard listing available courses.
func CourseListKeyboard(courses []course.CourseSummaryView) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true, OneTimeKeyboard: true}

	rows := make([]telebot.Row, 0, len(courses)+1) // +1 for the main menu button
	for _, c := range courses {
		btnText := c.PersianTitle
		rows = append(rows, menu.Row(menu.Text(btnText)))
	}
	rows = append(rows, menu.Row(BtnReturnToMainMenu))
	menu.Reply(rows...)
	return menu
}

// Constants for callback prefixes and button actions
const (
	CourseDetailsCallbackPrefix = "course_details:"
)

// CourseDetailsKeyboard generates reply keyboard for course overview.
func CourseDetailsKeyboard(courseID uint, progressPercentage int, isCompleted bool) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	var actionButtonText string
	if isCompleted {
		actionButtonText = ReviewCourseButtonText
	} else if progressPercentage > 0 {
		actionButtonText = fmt.Sprintf("%s (%d%%)", ContinueCourseButtonText, progressPercentage)
	} else {
		actionButtonText = StartCourseButtonText
	}

	menu.Reply(
		menu.Row(menu.Text(actionButtonText)),
		menu.Row(BtnReturnToMainMenu),
	)
	return menu
}

// InCourseNavigationKeyboard provides navigation within a course (e.g., next word).
func InCourseNavigationKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	menu.Reply(
		menu.Row(menu.Text(NextWordButtonText)),
		menu.Row(BtnReturnToMainMenu),
	)
	return menu
}

// CourseCompletedKeyboard shown when a course is finished.
func CourseCompletedKeyboard() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	menu.Reply(
		menu.Row(BtnReturnToMainMenu),
	)
	return menu
}

// Helper to generate callback data for selecting a course from an INLINE list
func CourseSelectionCallbackData(courseID uint) string {
	return fmt.Sprintf("%s%d", CourseDetailsCallbackPrefix, courseID)
}

// InlineCourseListKeyboard (Example if you were to use inline keyboards for course listing)
func InlineCourseListKeyboard(courses []course.CourseSummaryView) *telebot.ReplyMarkup {
	// Create a new ReplyMarkup for an inline keyboard.
	// The actual inline keyboard is built by adding telebot.InlineButton elements.
	markup := &telebot.ReplyMarkup{}

	var inlineKeyboardRows [][]telebot.InlineButton // A slice of button rows

	for _, c := range courses {
		btnText := c.PersianTitle
		if c.ProgressPercentage > 0 && !c.IsCompletedByUser {
			btnText = fmt.Sprintf("%s (%d%%)", btnText, c.ProgressPercentage)
		} else if c.IsCompletedByUser {
			btnText = fmt.Sprintf("%s (تکمیل شده)", btnText)
		}

		// Create an inline button
		inlineBtn := telebot.InlineButton{
			Unique: CourseDetailsCallbackPrefix + fmt.Sprintf("%d", c.ID), // Unique identifier for the button
			Text:   btnText,
			Data:   CourseSelectionCallbackData(c.ID), // Callback data
		}
		// Add the button as a new row (each button on its own row for this example)
		inlineKeyboardRows = append(inlineKeyboardRows, []telebot.InlineButton{inlineBtn})
	}

	// Set the InlineKeyboard field of the markup
	markup.InlineKeyboard = inlineKeyboardRows
	return markup
}

// Constants for course-related button texts that might be matched directly by text handlers.
const (
	StartCourseButtonText      = "شروع دوره"
	ContinueCourseButtonText   = "ادامه دوره"
	ReviewCourseButtonText     = "مرور دوره"
	NextWordButtonText         = "کلمه بعدی"
	BackToCourseListButtonText = "بازگشت به لیست دوره‌ها"
)
