package handlers

import (
	"errors"
	"fmt"
	"log"
	"strings" // Ensure strings is imported

	"github.com/2000ostd/enssi-tel-bot/internal/bot/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/services"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm" // For errors.Is(err, gorm.ErrRecordNotFound)
)

// HandleTextMessage is a general handler for text messages.
// It routes to more specific handlers based on the text content.
func HandleTextMessage(c telebot.Context, appServices *services.AppServices) error {
	userInput := c.Text()
	dbUser, err := appServices.User().GetOrCreateUserByTelegramID(
		c.Sender().ID,
		c.Sender().Username,
		c.Sender().FirstName,
		c.Sender().LastName,
	)
	if err != nil {
		return sendServiceError(c, "identifying user", err)
	}

	// Specific button texts that have their own direct handlers in router.go
	// are handled there first. This HandleTextMessage is a fallback.

	// Course Detail Action Buttons (e.g., "شروع دوره", "ادامه دوره (X%)", "مرور دوره")
	// These texts are dynamic, so we need to check prefixes or more stable identifiers.
	// We'll rely on the fact that selecting a course sets LastMenu to "course_details:<course_id>"
	// And then these buttons confirm the action for THAT course.
	if strings.HasPrefix(userInput, keyboards.StartCourseButtonText) ||
		strings.HasPrefix(userInput, keyboards.ContinueCourseButtonText) || // Matches "ادامه دوره" and "ادامه دوره (X%)"
		strings.HasPrefix(userInput, keyboards.ReviewCourseButtonText) {

		courseID, err := ParseIDFromState(dbUser.LastMenu, StateCourseDetailsPrefix)
		if err != nil {
			log.Printf("[HandleTextMessage] Start/Continue/Review Course: UserID %d, LastMenu '%s' not in 'course_details:ID' state or invalid courseID. Error: %v", dbUser.ID, dbUser.LastMenu, err)
			// Don't send error here, let it fall through or handle specific error if courseID is needed for other flows
		} else {
			// If courseID is valid, proceed to start/resume
			return handleStartOrResumeCourse(c, dbUser.ID, courseID, appServices)
		}
	}

	// Course Selection from List (User clicks a course title button)
	if strings.HasPrefix(dbUser.LastMenu, StateCourseList) {
		courses, listErr := appServices.Course().ListAvailableCourses(dbUser.ID)
		if listErr != nil {
			// Don't send error here, let it fall through
			log.Printf("[HandleTextMessage] Error listing courses for selection by title for UserID %d: %v", dbUser.ID, listErr)
		} else {
			for _, courseSummary := range courses {
				btnText := courseSummary.PersianTitle
				if courseSummary.Title != "" && courseSummary.Title != courseSummary.PersianTitle {
					btnText = fmt.Sprintf("%s (%s)", courseSummary.PersianTitle, courseSummary.Title)
				}
				if userInput == btnText {
					return displayCourseOverview(c, dbUser.ID, courseSummary.ID, appServices)
				}
			}
		}
	}

	log.Printf("[HandleTextMessage] Unhandled text from UserID %d: '%s', LastMenu: '%s'", dbUser.ID, userInput, dbUser.LastMenu)
	// Do not send "command not understood" here, as specific text handlers might have already run.
	// This is more of a fallback logger.
	return nil
}

// HandleStartLearningJourney is called when user clicks "Start Learning"
func HandleStartLearningJourney(c telebot.Context, userID uint, appServices *services.AppServices) error {
	err := appServices.User().UpdateUserLastMenu(userID, StateCourseList)
	if err != nil {
		log.Printf("[HandleStartLearningJourney] Error updating last menu for UserID %d: %v", userID, err)
	}

	courses, err := appServices.Course().ListAvailableCourses(userID)
	if err != nil {
		return sendServiceError(c, "listing available courses", err)
	}

	if len(courses) == 0 {
		return c.Send("متاسفانه در حال حاضر دوره‌ای برای نمایش وجود ندارد.", keyboards.MainMenu)
	}

	desc := ""
	if desc == "" {
		desc = "لطفا یک دوره را برای شروع انتخاب کنید:"
	}
	return c.Send(desc, keyboards.CourseListKeyboard(courses), telebot.ModeMarkdownV2)
}

// displayCourseOverview shows details of a selected course. (Can remain unexported if only called by HandleTextMessage)
func displayCourseOverview(c telebot.Context, userID uint, courseID uint, appServices *services.AppServices) error {
	err := appServices.User().UpdateUserLastMenu(userID, CourseDetailsMenuState(courseID))
	if err != nil {
		log.Printf("[displayCourseOverview] Error updating last menu for UserID %d: %v", userID, err)
	}

	courseOverview, err := appServices.Course().GetCourseOverview(courseID, userID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Send("دوره مورد نظر یافت نشد.", keyboards.MainMenu)
		}
		return sendServiceError(c, fmt.Sprintf("getting overview for course %d", courseID), err)
	}

	formattedOverview := formatters.FormatCourseOverview(courseOverview)
	return c.Send(
		formattedOverview,
		keyboards.CourseDetailsKeyboard(courseID, courseOverview.ProgressPercentage, courseOverview.IsCompletedByUser),
		telebot.ModeMarkdownV2,
	)
}

// handleStartOrResumeCourse (Can remain unexported if only called by HandleTextMessage)
func handleStartOrResumeCourse(c telebot.Context, userID uint, courseID uint, appServices *services.AppServices) error {
	log.Printf("[handleStartOrResumeCourse] UserID %d attempting to start/resume CourseID %d", userID, courseID)
	learningContext, err := appServices.Course().StartOrResumeLearningSession(courseID, userID)
	if err != nil {
		return sendServiceError(c, fmt.Sprintf("starting/resuming course %d", courseID), err)
	}
	return sendLearningContext(c, learningContext, appServices)
}

// HandleAdvanceWord is called when user clicks "Next Word".
func HandleAdvanceWord(c telebot.Context, userID uint, courseID uint, appServices *services.AppServices) error {
	log.Printf("[HandleAdvanceWord] UserID %d advancing in CourseID %d", userID, courseID)

	_, err := appServices.Course().GetUserCourse(userID, courseID)
	if err != nil {
		log.Printf("[HandleAdvanceWord] User %d not actively in course %d or error: %v. Sending to main menu.", userID, courseID, err)
		appServices.User().UpdateUserLastMenu(userID, StateMain) // Use exported constant
		return c.Send("به نظر می‌رسد از دوره خارج شده‌اید. لطفا دوباره آن را از لیست دوره‌ها انتخاب کنید.", keyboards.MainMenu)
	}

	learningContext, err := appServices.Course().AdvanceToNextWord(courseID, userID)
	if err != nil {
		log.Printf("[HandleAdvanceWord] Error advancing word for UserID %d in CourseID %d: %v", userID, courseID, err)
		if learningContext == nil {
			return sendServiceError(c, fmt.Sprintf("advancing to next word in course %d", courseID), err)
		}
	}
	return sendLearningContext(c, learningContext, appServices)
}

// HandleReturnToMainMenu attempts to clean up active quiz messages.
func HandleReturnToMainMenu(c telebot.Context, userID uint, appServices *services.AppServices) error {
	log.Printf("[HandleReturnToMainMenu] UserID %d returning to main menu.", userID)

	activeAttempt, err := appServices.Quiz().FindAnyActiveQuizAttempt(userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[HandleReturnToMainMenu] UserID %d: Error finding active quiz: %v", userID, err)
	}

	if activeAttempt != nil && activeAttempt.CurrentQuestionMessageID != 0 {
		log.Printf("[HandleReturnToMainMenu] UserID %d: Active QuizAttemptID: %d, MsgID: %d. Deleting message.",
			userID, activeAttempt.ID, activeAttempt.CurrentQuestionMessageID)

		messageToDelete := telebot.StoredMessage{
			MessageID: fmt.Sprint(activeAttempt.CurrentQuestionMessageID),
			ChatID:    c.Chat().ID,
		}
		if errDel := c.Bot().Delete(messageToDelete); errDel != nil {
			log.Printf("[HandleReturnToMainMenu] UserID %d: Failed to delete quiz message (MsgID: %s): %v",
				userID, messageToDelete.MessageID, errDel)
		} else {
			log.Printf("[HandleReturnToMainMenu] UserID %d: Successfully deleted quiz message (MsgID: %s).", userID, messageToDelete.MessageID)
		}
	}

	err = appServices.User().UpdateUserLastMenu(userID, StateMain) // Use exported constant
	if err != nil {
		log.Printf("[HandleReturnToMainMenu] UserID %d: Error updating LastMenu: %v", userID, err)
	}

	returnText := ""
	if returnText == "" {
		returnText = "به منوی اصلی بازگشتید"
	}
	return c.Send(returnText, keyboards.MainMenu, telebot.ModeMarkdownV2)
}
