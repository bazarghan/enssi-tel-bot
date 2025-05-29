package handlers

import (
	"errors" // For gorm.ErrRecordNotFound
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
	// "math" // For math.Round if needed, but int conversion is usually fine for display
)

func handleCourseSelection(ctx telebot.Context, db *gorm.DB) error {
	label := ctx.Text() // This is PersianTitle of the course

	if label == "بازگشت به منوی اصلی" { // Make sure this text matches your actual button
		return handleReturnToMainMenu(ctx, db)
	}

	var course models.Course
	if err := db.Where("persian_title = ?", label).First(&course).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.Send("متاسفانه دوره‌ای با این عنوان یافت نشد.")
		}
		fmt.Printf("Error fetching course by persian_title '%s': %v\n", label, err)
		return ctx.Send("مشکلی در یافتن دوره پیش آمد.")
	}

	user, err := fetchUser(ctx, db) // Assuming fetchUser gets the current user
	if err != nil {
		return ctx.Send("مشکلی در یافتن اطلاعات کاربری شما پیش آمد.")
	}

	// Set LastMenu to indicate user is viewing details for this course
	user.LastMenu = fmt.Sprintf("courseId:%d", course.ID)
	if err := db.Save(user).Error; err != nil {
		fmt.Printf("Error saving user LastMenu: %v\n", err)
		return ctx.Send("خطایی در به‌روزرسانی وضعیت شما رخ داد.")
	}
	// --- Logic to determine progress for button text ---
	var userCourse models.UserCourse
	progressPercentage := -1 // Default: -1 indicates not started or no progress record

	errUserCourse := db.Where("user_id = ? AND course_id = ?", user.ID, course.ID).First(&userCourse).Error

	if errUserCourse == nil { // UserCourse record exists
		if userCourse.Progress > 0 { // User has made some progress
			var totalCourseWords int64
			// Count total words (CourseWord entries) for this course.
			// This assumes each CourseWord is a learnable unit.
			// If CourseWord has an 'Index' field, MAX(Index) might be more accurate for "course length".
			if errCount := db.Model(&models.CourseWord{}).Where("course_id = ?", course.ID).Count(&totalCourseWords).Error; errCount != nil {
				fmt.Printf("Error counting total words for course %d: %v\n", course.ID, errCount)
				// Cannot calculate percentage, keep progressPercentage as -1 or 0 if userCourse.Progress > 0
				// Let's treat as "Continue Course" without percentage if count fails but progress exists.
				progressPercentage = 0 // Or a special value to indicate "Continue" without %
				// For simplicity, if count fails but progress > 0, we can show 0% or just "Continue"
				// The keyboard expects an int. -1 -> "Start", >=0 -> "Continue X%"
				// So if count fails, and progress > 0, we might still want to show "Continue"
				// Let's default to showing "Start Course" if total words can't be determined.
				// Or, if userCourse.Progress > 0, we can ensure progressPercentage is at least 0.
			}

			if totalCourseWords > 0 {
				currentProgress := userCourse.Progress
				// Cap progress at totalCourseWords for calculation to avoid >100% if data is inconsistent
				if currentProgress > uint(totalCourseWords) {
					currentProgress = uint(totalCourseWords)
				}
				progressPercentage = int((float64(currentProgress) / float64(totalCourseWords)) * 100)
			} else if userCourse.Progress > 0 {
				// Has progress, but no words found for course (data issue). Show 0% or a plain "Continue".
				// Setting to 0 will make the button "ادامه دوره - 0%"
				progressPercentage = 0
			}
			// If userCourse.Progress is 0, progressPercentage remains -1 (from initialization)
			// which will lead to "شروع دوره", which is fine.
		}
		// If userCourse.Progress is 0, progressPercentage remains -1, leading to "شروع دوره"
	}
	// If errUserCourse is gorm.ErrRecordNotFound, progressPercentage remains -1.

	desc := course.PersianDescription
	return ctx.Send(
		desc,
		keyboards.CourseDetails(progressPercentage), // Pass the calculated percentage
	)
}
