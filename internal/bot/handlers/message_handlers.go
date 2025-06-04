package handlers

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/bot/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/services"
	"github.com/2000ostd/enssi-tel-bot/internal/services/course"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

// HandleTextMessage is a general handler for text messages.
// It routes to more specific handlers based on the text content or user state.
func HandleTextMessage(c telebot.Context, appServices *services.AppServices) error {
	userInput := c.Text()
	dbUser, err := appServices.User().GetOrCreateUserByTelegramID(
		c.Sender().ID,
		c.Sender().Username,
		c.Sender().FirstName,
		c.Sender().LastName,
	)
	if err != nil {
		return sendServiceError(c, "identifying user (text message)", err)
	}

	// Check for mandatory daily review FIRST for any text message interaction
	reviewHandled, reviewErr := CheckAndInitiateReview(c, appServices, dbUser)
	if reviewErr != nil {
		return sendServiceError(c, "checking for daily review (text message)", reviewErr)
	}
	if reviewHandled {
		return nil // Review process has taken over
	}

	// --- Specific button texts that have their own direct handlers in router.go ---
	// (e.g., "شروع یادگیری", "پروفایل", "بازگشت به منوی اصلی", "کلمه بعدی")
	// These are handled by the router directly. If a message matches one of those,
	// this HandleTextMessage might not even be called, or if it is, those texts
	// should not be re-processed here. The router handles them first.

	// This handler is more for text inputs that are NOT predefined keyboard buttons
	// handled by specific b.Handle(keyboards.BtnXYZ.Text, ...) routes.
	// Example: User types a course name from a list.

	// Course Detail Action Buttons (e.g., "شروع دوره", "ادامه دوره (X%)", "مرور دوره")
	// These rely on user.LastMenu being "course_details:<course_id>"
	if strings.HasPrefix(dbUser.LastMenu, StateCourseDetailsPrefix) {
		courseID, err := ParseIDFromState(dbUser.LastMenu, StateCourseDetailsPrefix)
		if err == nil { // Successfully parsed courseID from state
			switch {
			// Match against the dynamic parts of button texts
			case strings.HasPrefix(userInput, keyboards.StartCourseButtonText): // "شروع دوره"
				return handleStartOrResumeCourse(c, dbUser.ID, courseID, appServices)
			case strings.HasPrefix(userInput, keyboards.ContinueCourseButtonText): // "ادامه دوره" (catches "ادامه دوره (X%)")
				return handleStartOrResumeCourse(c, dbUser.ID, courseID, appServices)
			case strings.HasPrefix(userInput, keyboards.ReviewCourseButtonText): // "مرور دوره"
				// TODO: Implement actual course review logic (e.g., different type of quiz or content)
				log.Printf("[HandleTextMessage] UserID %d selected 'Review Course' for CourseID %d. Not yet implemented.", dbUser.ID, courseID)
				return c.Send("مرور دوره هنوز پیاده‌سازی نشده است.", keyboards.CourseDetailsKeyboard(courseID, 0, false)) // Placeholder
			}
		} else {
			log.Printf("[HandleTextMessage] UserID %d, LastMenu '%s' implies course details, but failed to parse CourseID. Input: '%s'", dbUser.ID, dbUser.LastMenu, userInput)
		}
	}

	// Course Selection from List (User clicks a course title button from a ReplyKeyboard)
	// This assumes LastMenu is StateCourseList
	if dbUser.LastMenu == StateCourseList {
		courses, listErr := appServices.Course().ListAvailableCourses(dbUser.ID)
		if listErr != nil {
			log.Printf("[HandleTextMessage] Error listing courses for selection by title for UserID %d: %v", dbUser.ID, listErr)
			// Don't send error to user here, might be an unhandled text.
		} else {
			for _, courseSummary := range courses {
				// Construct the button text exactly as it appears on the keyboard
				// This needs to precisely match how CourseListKeyboard generates button texts.
				btnText := courseSummary.PersianTitle
				// Assuming CourseListKeyboard doesn't add "(Title)" if PersianTitle and Title are different for reply keyboards.
				// If it does, this match needs to be identical.
				// For simplicity, let's assume it's just PersianTitle for reply keyboard buttons.
				if userInput == btnText {
					return displayCourseOverview(c, dbUser.ID, courseSummary.ID, appServices)
				}
			}
		}
	}

	// If no specific action is matched:
	log.Printf("[HandleTextMessage] Unhandled text from UserID %d: '%s', LastMenu: '%s'", dbUser.ID, userInput, dbUser.LastMenu)
	// It's often better not to send "Command not understood" for every unhandled text,
	// as it might be part of a multi-step interaction or a typo.
	// Consider sending a generic help message or main menu if truly unhandled.
	// For now, just log.
	return nil
}

// HandleStartLearningJourney is called when user clicks "شروع یادگیری"
// This function itself is usually called from the router, not from HandleTextMessage.
// The review check should be in the router's handler for this button.
func HandleStartLearningJourney(c telebot.Context, userID uint, appServices *services.AppServices) error {
	// NOTE: The CheckAndInitiateReview should ideally be done *before* this handler is called,
	// typically in the router or a middleware. If it's here, it means the router didn't do it.
	// For consistency, let's assume the router or a preceding general handler does it.
	// If not, it would be:
	/*
		dbUser, _ := appServices.User().GetOrCreateUserByTelegramID(c.Sender().ID, c.Sender().Username, c.Sender().FirstName, c.Sender().LastName)
		reviewHandled, reviewErr := CheckAndInitiateReview(c, appServices, dbUser)
		if reviewErr != nil { return sendServiceError(c, "checking review", reviewErr) }
		if reviewHandled { return nil }
	*/

	err := appServices.User().UpdateUserLastMenu(userID, StateCourseList)
	if err != nil {
		log.Printf("[HandleStartLearningJourney] Error updating last menu for UserID %d: %v", userID, err)
		// Non-fatal for listing courses
	}

	courses, err := appServices.Course().ListAvailableCourses(userID)
	if err != nil {
		return sendServiceError(c, "listing available courses", err)
	}

	if len(courses) == 0 {
		return c.Send("متاسفانه در حال حاضر دوره‌ای برای نمایش وجود ندارد.", keyboards.MainMenu)
	}

	desc := "لطفا یک دوره را برای شروع انتخاب کنید:"
	return c.Send(desc, keyboards.CourseListKeyboard(courses), telebot.ModeMarkdownV2)
}

// displayCourseOverview shows details of a selected course.
func displayCourseOverview(c telebot.Context, userID uint, courseID uint, appServices *services.AppServices) error {
	// Review check should happen before calling this if it's a user-initiated action.
	// If called from HandleTextMessage, HandleTextMessage already did the check.

	err := appServices.User().UpdateUserLastMenu(userID, CourseDetailsMenuState(courseID))
	if err != nil {
		log.Printf("[displayCourseOverview] Error updating last menu for UserID %d: %v", userID, err)
	}

	courseOverview, err := appServices.Course().GetCourseOverview(courseID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, course.ErrCourseNotFound) {
			return c.Send("دوره مورد نظر یافت نشد.", keyboards.MainMenu)
		}
		return sendServiceError(c, fmt.Sprintf("getting overview for course %d", courseID), err)
	}

	formattedOverview := formatters.FormatCourseOverview(courseOverview)
	// ProgressPercentage and IsCompletedByUser are now on courseOverview directly
	return c.Send(
		formattedOverview,
		keyboards.CourseDetailsKeyboard(courseID, courseOverview.ProgressPercentage, courseOverview.IsCompletedByUser),
		telebot.ModeMarkdownV2,
	)
}

// handleStartOrResumeCourse starts or resumes a course.
func handleStartOrResumeCourse(c telebot.Context, userID uint, courseID uint, appServices *services.AppServices) error {
	// Review check should happen before calling this.
	// If called from HandleTextMessage, HandleTextMessage already did the check.

	log.Printf("[handleStartOrResumeCourse] UserID %d attempting to start/resume CourseID %d", userID, courseID)
	learningContext, err := appServices.Course().StartOrResumeLearningSession(courseID, userID)
	if err != nil {
		// Specific error handling for course not found or other issues
		if errors.Is(err, course.ErrCourseNotFound) {
			return c.Send("دوره مورد نظر برای شروع یافت نشد.", keyboards.MainMenu)
		}
		return sendServiceError(c, fmt.Sprintf("starting/resuming course %d", courseID), err)
	}
	return sendLearningContext(c, learningContext, appServices)
}

// HandleAdvanceWord is called when user clicks "کلمه بعدی"
// The review check should be in the router's handler for this button.
func HandleAdvanceWord(c telebot.Context, userID uint, courseID uint, appServices *services.AppServices) error {
	// Review check should be done by the router before calling this.
	log.Printf("[HandleAdvanceWord] UserID %d advancing in CourseID %d", userID, courseID)

	// Ensure user is actually in the course (basic check, UserCourse state is more robust)
	_, err := appServices.Course().GetUserCourse(userID, courseID)
	if err != nil {
		log.Printf("[HandleAdvanceWord] User %d not actively in course %d or error: %v. Sending to main menu.", userID, courseID, err)
		appServices.User().UpdateUserLastMenu(userID, StateMain)
		return c.Send("به نظر می‌رسد از دوره خارج شده‌اید. لطفا دوباره آن را از لیست دوره‌ها انتخاب کنید.", keyboards.MainMenu)
	}

	learningContext, err := appServices.Course().AdvanceToNextWord(courseID, userID)
	if err != nil {
		log.Printf("[HandleAdvanceWord] Error advancing word for UserID %d in CourseID %d: %v", userID, courseID, err)
		// If learningContext is nil, send a generic service error.
		// If learningContext is not nil but there was an error (e.g., word formatting failed but context exists),
		// sendLearningContext might still be able to show something or an error message from the context.
		if learningContext == nil {
			return sendServiceError(c, fmt.Sprintf("advancing to next word in course %d", courseID), err)
		}
	}
	// sendLearningContext will handle nil lc or lc with error messages.
	return sendLearningContext(c, learningContext, appServices)
}

// HandleReturnToMainMenu attempts to clean up active quiz messages.
// The review check is less critical here as it's an explicit "exit" action,
// but for consistency, if other main menu actions have it, this could too.
// However, if a review is pending, returning to main menu should still be allowed.
// So, NO CheckAndInitiateReview here.
func HandleReturnToMainMenu(c telebot.Context, userID uint, appServices *services.AppServices) error {
	log.Printf("[HandleReturnToMainMenu] UserID %d returning to main menu.", userID)

	// Find any active quiz (course or review) to potentially clean up its message
	activeAttempt, err := appServices.Quiz().FindAnyActiveQuizAttempt(userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) { // Log DB errors, ignore not found
		log.Printf("[HandleReturnToMainMenu] UserID %d: Error finding active quiz: %v", userID, err)
	}

	if activeAttempt != nil && activeAttempt.CurrentQuestionMessageID != 0 {
		log.Printf("[HandleReturnToMainMenu] UserID %d: Active QuizAttemptID: %d, MsgID: %d. Attempting to delete message.",
			userID, activeAttempt.ID, activeAttempt.CurrentQuestionMessageID)

		// To delete, we need the chat ID. c.Chat() should provide it.
		if c.Chat() == nil {
			log.Printf("[HandleReturnToMainMenu] UserID %d: Cannot delete quiz message, chat context is nil.", userID)
		} else {
			messageToDelete := telebot.StoredMessage{
				MessageID: strconv.Itoa(activeAttempt.CurrentQuestionMessageID), // MessageID needs to be string
				ChatID:    c.Chat().ID,
			}
			// Edit to something neutral and remove keyboard instead of deleting,
			// as deleting might fail or look abrupt.
			neutralText := "آزمون لغو شد."
			editPreviousMessage(c, messageToDelete.ChatID, messageToDelete.MessageID, neutralText, &telebot.ReplyMarkup{RemoveKeyboard: true})
			log.Printf("[HandleReturnToMainMenu] UserID %d: Edited active quiz message (MsgID: %s) to neutral and removed keyboard.", userID, messageToDelete.MessageID)

			// Optionally, you might want to mark the quiz attempt as "abandoned" or reset its CurrentQuestionMessageID
			// For now, just editing the message.
		}
	}

	err = appServices.User().UpdateUserLastMenu(userID, StateMain)
	if err != nil {
		log.Printf("[HandleReturnToMainMenu] UserID %d: Error updating LastMenu: %v", userID, err)
	}

	returnText := "به منوی اصلی بازگشتید."
	return c.Send(returnText, keyboards.MainMenu, telebot.ModeMarkdownV2)
}
