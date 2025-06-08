package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/bot/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"github.com/2000ostd/enssi-tel-bot/internal/services"
	"github.com/2000ostd/enssi-tel-bot/internal/services/course"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

// HandleStateBasedText is the primary state machine for all user text input.
// It routes user input to the correct function based on their `LastMenu` state.
func HandleStateBasedText(c telebot.Context, dbUser *models.User, appServices *services.AppServices) error {
	userInput := strings.TrimSpace(c.Text())
	// No need to fetch the user again, it's passed as a parameter.
	if dbUser == nil {
		return SendServiceError(c, "processing text (state handler)", errors.New("user object is nil"))
	}

	// --- Global Text Commands ---
	// Handle commands that should work regardless of the current state, like returning to the main menu.
	if userInput == keyboards.BtnReturnToMainMenu.Text {
		return HandleReturnToMainMenu(c, dbUser.ID, appServices)
	}

	// --- Daily Review Check ---
	// Check for a mandatory review before processing state-specific actions,
	// unless the user is already in a quiz or the admin panel.
	isProtectedState := strings.HasPrefix(dbUser.LastMenu, StateInQuizPrefix) ||
		strings.HasPrefix(dbUser.LastMenu, StateInReviewQuizPrefix) ||
		dbUser.LastMenu == StateInAdminPanel

	if !isProtectedState {
		reviewHandled, reviewErr := CheckAndInitiateReview(c, appServices, dbUser)
		if reviewErr != nil {
			return SendServiceError(c, "checking for daily review (state handler)", reviewErr)
		}
		if reviewHandled {
			return nil // Review process has taken over. Stop further processing.
		}
	}

	// --- State Machine Dispatcher ---
	log.Printf("[State Machine] Routing UserID %d from state '%s' with input: '%s'", dbUser.ID, dbUser.LastMenu, userInput)

	switch {
	case dbUser.LastMenu == StateMain:
		// User is in the main menu.
		switch userInput {
		case keyboards.BtnStartLearning.Text:
			return HandleStartLearningJourney(c, dbUser.ID, appServices)
		case keyboards.BtnMyProfile.Text:
			return HandleMyProfileCommand(c, appServices) // The state check is now inside this handler.
		case keyboards.BtnAdminPanel.Text:
			return HandleAdminPanelEntry(c, dbUser, appServices) // New dedicated handler for clarity
		}

	case dbUser.LastMenu == StateCourseList:
		// User is viewing the list of courses. Input should be a course title.
		return handleCourseSelectionFromList(c, userInput, dbUser.ID, appServices)

	case dbUser.LastMenu == StateInAdminPanel:
		// User is in the admin panel. Input is an SQL query.
		return handleAdminQuery(c, userInput, dbUser, appServices)

	case strings.HasPrefix(dbUser.LastMenu, StateCourseDetailsPrefix):
		// User is viewing a specific course overview.
		courseID, err := ParseIDFromState(dbUser.LastMenu, StateCourseDetailsPrefix)
		if err != nil {
			log.Printf("[State Machine] Could not parse courseID from state '%s' for UserID %d", dbUser.LastMenu, dbUser.ID)
			return SendServiceError(c, "invalid state", err)
		}
		// Check for buttons like "Start Course" or "Continue Course"
		if strings.HasPrefix(userInput, keyboards.StartCourseButtonText) || strings.HasPrefix(userInput, keyboards.ContinueCourseButtonText) {
			return handleStartOrResumeCourse(c, dbUser.ID, courseID, appServices)
		}
		if strings.HasPrefix(userInput, keyboards.ReviewCourseButtonText) {
			return c.Send("مرور دوره هنوز پیاده‌سازی نشده است.") // Placeholder
		}

	case strings.HasPrefix(dbUser.LastMenu, StateInCoursePrefix):
		// User is actively in a course lesson.
		if userInput == keyboards.NextWordButtonText {
			courseID, err := ParseIDFromState(dbUser.LastMenu, StateInCoursePrefix)
			if err != nil {
				log.Printf("[State Machine] Could not parse courseID from state '%s' for UserID %d", dbUser.LastMenu, dbUser.ID)
				return SendServiceError(c, "invalid state", err)
			}
			return HandleAdvanceWord(c, dbUser.ID, courseID, appServices)
		}

	case strings.HasPrefix(dbUser.LastMenu, StateInQuizPrefix) || strings.HasPrefix(dbUser.LastMenu, StateInReviewQuizPrefix):
		// User is in a quiz. Inline keyboard handles answers. Text messages should be ignored or handled.
		return c.Send("لطفا با استفاده از دکمه‌ها به سوالات آزمون پاسخ دهید.")
	}

	// If no state matches, log it. Don't send a message to avoid spamming on typos.
	log.Printf("[State Machine] UserID %d sent unhandled text '%s' from state '%s'", dbUser.ID, userInput, dbUser.LastMenu)
	return nil
}

// --- Helper Handlers for the State Machine ---

func handleCourseSelectionFromList(c telebot.Context, userInput string, userID uint, appServices *services.AppServices) error {
	courses, err := appServices.Course().ListAvailableCourses(userID)
	if err != nil {
		return SendServiceError(c, "listing courses for selection", err)
	}
	for _, courseSummary := range courses {
		// Match against the exact button text generated by the keyboard
		if userInput == courseSummary.PersianTitle {
			return displayCourseOverview(c, userID, courseSummary.ID, appServices)
		}
	}
	// If no match found, it might be a typo. Silently ignore.
	return nil
}

func HandleAdminPanelEntry(c telebot.Context, dbUser *models.User, appServices *services.AppServices) error {
	if !dbUser.IsAdmin {
		log.Printf("[Admin] Non-admin UserID %d attempted to access admin panel.", dbUser.ID)
		return c.Send("شما اجازه دسترسی به این بخش را ندارید.")
	}
	err := appServices.User().UpdateUserLastMenu(dbUser.ID, StateInAdminPanel)
	if err != nil {
		return SendServiceError(c, "entering admin panel", err)
	}
	adminPanelMessage := "به پنل ادمین خوش آمدید. 👨‍💻\n" +
		"اکنون می‌توانید کوئری‌های SQL خام را مستقیماً ارسال کنید.\n" +
		"هشدار: اجرای کوئری‌های نادرست می‌تواند به داده‌ها آسیب برساند.\n" +
		"برای خروج، از دکمه 'بازگشت به منوی اصلی' استفاده کنید."

	return c.Send(formatters.EscapeMarkdownV2(adminPanelMessage), keyboards.BackToMainMenuKeyboard(), telebot.ModeMarkdownV2)
}

func handleAdminQuery(c telebot.Context, query string, dbUser *models.User, appServices *services.AppServices) error {
	log.Printf("[Admin] UserID %d executing SQL query: %s", dbUser.ID, query)
	// Simple guards against common destructive commands
	lowerQuery := strings.ToLower(query)
	if (strings.HasPrefix(lowerQuery, "delete ") || strings.HasPrefix(lowerQuery, "update ")) && !strings.Contains(lowerQuery, "where") {
		return c.Send(formatters.EscapeMarkdownV2("⚠️ هشدار: کوئری‌های DELETE/UPDATE بدون WHERE مجاز نیستند."), telebot.ModeMarkdownV2)
	}
	if strings.HasPrefix(lowerQuery, "drop ") || strings.HasPrefix(lowerQuery, "truncate ") {
		return c.Send(formatters.EscapeMarkdownV2("⛔️ دستورات DROP و TRUNCATE از طریق این پنل مجاز نیستند."), telebot.ModeMarkdownV2)
	}

	var results []map[string]interface{}
	var responseMessage string
	db := appServices.DB()
	tx := db.Raw(query).Scan(&results)

	if tx.Error != nil {
		log.Printf("[Admin] SQL Error for UserID %d: %v", dbUser.ID, tx.Error)
		// The error message itself is escaped and placed within a code block. The prefix is safe.
		responseMessage = fmt.Sprintf("❌ خطای SQL:\n```\n%s\n```", formatters.EscapeMarkdownV2(tx.Error.Error()))
	} else {
		if len(results) > 0 {
			jsonResult, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				// Escape the entire message as it's just plain text.
				messagePart := fmt.Sprintf("✅ کوئری اجرا شد. %d ردیف تحت تاثیر. نمایش نتیجه با خطا مواجه شد: %v", tx.RowsAffected, err)
				responseMessage = formatters.EscapeMarkdownV2(messagePart)
			} else {
				resultStr := string(jsonResult)
				if len(resultStr) > 4000 {
					resultStr = resultStr[:4000] + "\n... (نتیجه طولانی‌تر از حد مجاز است)"
				}
				// --- Selective Escaping ---
				// 1. Create the prefix text.
				prefixText := fmt.Sprintf("✅ کوئری اجرا شد. %d ردیف بازگردانده شد.\nنتیجه:\n", len(results))
				// 2. Escape ONLY the prefix text.
				escapedPrefix := formatters.EscapeMarkdownV2(prefixText)
				// 3. Combine with the unescaped JSON code block.
				responseMessage = escapedPrefix + "```json\n" + resultStr + "\n```"
			}
		} else if tx.RowsAffected > 0 {
			// Escape the entire message as it's just plain text.
			messagePart := fmt.Sprintf("✅ کوئری اجرا شد. %d ردیف تحت تاثیر قرار گرفت.", tx.RowsAffected)
			responseMessage = formatters.EscapeMarkdownV2(messagePart)
		} else {
			// Escape the entire message as it's just plain text.
			responseMessage = formatters.EscapeMarkdownV2("✅ کوئری اجرا شد. هیچ ردیفی بازگردانده نشد یا تحت تاثیر قرار نگرفت.")
		}
	}
	// Send the final, correctly constructed message.
	return c.Send(responseMessage, telebot.ModeMarkdownV2)
}

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
		return SendServiceError(c, "identifying user (text message)", err)
	}

	// ---> ADD SQL EXECUTION LOGIC FOR ADMIN PANEL STATE <---
	if dbUser.LastMenu == StateInAdminPanel {
		if !dbUser.IsAdmin { // Double check admin status
			log.Printf("[HandleTextMessage-Admin] Non-admin UserID %d in admin state. Resetting.", dbUser.ID)
			appServices.User().UpdateUserLastMenu(dbUser.ID, StateMain)
			return c.Send("خطای دسترسی. به منوی اصلی بازگشتید.", keyboards.NewMainMenu(false))
		}

		log.Printf("[HandleTextMessage-Admin] UserID %d (Admin) executing SQL query: %s", dbUser.ID, userInput)

		// Basic safety: disallow very common destructive commands if sent plainly.
		// This is NOT a robust security measure but a simple guard.
		lowerInput := strings.ToLower(userInput)
		if strings.HasPrefix(lowerInput, "drop ") || strings.HasPrefix(lowerInput, "delete from ") || strings.HasPrefix(lowerInput, "truncate ") {
			if !strings.Contains(lowerInput, "where") && (strings.HasPrefix(lowerInput, "delete from ") || strings.HasPrefix(lowerInput, "update ")) {
				// Basic check for DELETE/UPDATE without WHERE, still very rudimentary
				c.Send(formatters.EscapeMarkdownV2("⚠️ هشدار: کوئری‌های DELETE/UPDATE بدون WHERE بسیار خطرناک هستند. با احتیاط ادامه دهید."), telebot.ModeMarkdownV2, keyboards.BackToMainMenuKeyboard())
				// return nil // Or let it proceed if admin confirms
			} else if strings.HasPrefix(lowerInput, "drop ") || strings.HasPrefix(lowerInput, "truncate ") {
				return c.Send(formatters.EscapeMarkdownV2("⛔️ دستورات DROP و TRUNCATE از طریق این پنل مجاز نیستند."), telebot.ModeMarkdownV2, keyboards.BackToMainMenuKeyboard())
			}
		}

		var results []map[string]interface{}
		var rawSQLMessage string

		// Use the AppServices DB accessor
		db := appServices.DB()
		tx := db.Raw(userInput).Scan(&results) // .Scan works well for SELECT returning rows

		// ... inside the admin panel logic in HandleTextMessage ...
		if tx.Error != nil {
			log.Printf("[HandleTextMessage-Admin] SQL Error for UserID %d: %v", dbUser.ID, tx.Error)
			// Error message part: The error itself IS escaped. The prefix "❌ خطای SQL:\n" is static and safe.
			rawSQLMessage = fmt.Sprintf("❌ خطای SQL:\n```\n%s\n```", formatters.EscapeMarkdownV2(tx.Error.Error()))
		} else {
			if len(results) > 0 {
				jsonResult, err := json.MarshalIndent(results, "", "  ")
				if err != nil {
					// This message needs its static parts escaped
					messagePart := fmt.Sprintf("✅ کوئری اجرا شد. %d ردیف تحت تاثیر. نمایش نتیجه با خطا مواجه شد: %v", tx.RowsAffected, err)
					rawSQLMessage = formatters.EscapeMarkdownV2(messagePart)
				} else {
					resultStr := string(jsonResult)
					if len(resultStr) > 4000 {
						resultStr = resultStr[:4000] + "\n... (نتیجه طولانی‌تر از حد مجاز است)"
					}
					// This is the key part: The text OUTSIDE the json block needs escaping
					// if it contains special chars like '.'
					prefixText := fmt.Sprintf("✅ کوئری اجرا شد. %d ردیف تحت تاثیر / بازگردانده شد.\nنتیجه:\n", tx.RowsAffected)
					escapedPrefixText := formatters.EscapeMarkdownV2(prefixText)
					rawSQLMessage = escapedPrefixText + "```json\n" + resultStr + "\n```"
				}
			} else if tx.RowsAffected > 0 {
				// This message needs its static parts escaped
				messagePart := fmt.Sprintf("✅ کوئری اجرا شد. %d ردیف تحت تاثیر قرار گرفت.", tx.RowsAffected)
				rawSQLMessage = formatters.EscapeMarkdownV2(messagePart)
			} else {
				// This static message needs escaping
				rawSQLMessage = formatters.EscapeMarkdownV2("✅ کوئری اجرا شد. هیچ ردیفی بازگردانده نشد یا تحت تاثیر قرار نگرفت.")
			}
		}
		// Now send rawSQLMessage WITHOUT an overall escape, because parts of it are already escaped
		// and the JSON part is intentionally not.
		return c.Send(rawSQLMessage, telebot.ModeMarkdownV2, keyboards.BackToMainMenuKeyboard())
	}
	// ---> END OF SQL EXECUTION LOGIC <---

	// Check for mandatory daily review FIRST for any text message interaction
	reviewHandled, reviewErr := CheckAndInitiateReview(c, appServices, dbUser)
	if reviewErr != nil {
		return SendServiceError(c, "checking for daily review (text message)", reviewErr)
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
				return c.Send("مرور دوره هنوز پیاده‌سازی نشده است.", keyboards.CourseDetailsKeyboard(courseID, 0, true, false)) // Placeholder
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
		return SendServiceError(c, "listing available courses", err)
	}

	if len(courses) == 0 {
		return c.Send("متاسفانه در حال حاضر دوره‌ای برای نمایش وجود ندارد.", keyboards.MainMenu)
	}

	desc := "لطفا یک دوره را برای شروع انتخاب کنید:"
	return c.Send(desc, keyboards.CourseListKeyboard(courses), telebot.ModeMarkdownV2)
}

// displayCourseOverview shows details of a selected course.
func displayCourseOverview(
	c telebot.Context,
	userID uint,
	courseID uint,
	appServices *services.AppServices,
) error {
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
		return SendServiceError(c, fmt.Sprintf("getting overview for course %d", courseID), err)
	}

	formattedText := formatters.FormatCourseOverview(courseOverview)
	if courseOverview.IsStartedByUser {
		formattedText = formatters.FormatCourseProgress(courseOverview)
	}

	// ProgressPercentage and IsCompletedByUser are now on courseOverview directly
	return c.Send(
		formattedText,
		keyboards.CourseDetailsKeyboard(
			courseID,
			courseOverview.ProgressPercentage,
			courseOverview.IsStartedByUser,
			courseOverview.IsCompletedByUser),
		telebot.ModeMarkdownV2,
	)
}

// handleStartOrResumeCourse starts or resumes a course.
func handleStartOrResumeCourse(
	c telebot.Context,
	userID uint,
	courseID uint,
	appServices *services.AppServices,
) error {
	// Review check should happen before calling this.
	// If called from HandleTextMessage, HandleTextMessage already did the check.

	log.Printf("[handleStartOrResumeCourse] UserID %d attempting to start/resume CourseID %d", userID, courseID)
	learningContext, err := appServices.Course().StartOrResumeLearningSession(courseID, userID)
	if err != nil {
		// Specific error handling for course not found or other issues
		if errors.Is(err, course.ErrCourseNotFound) {
			return c.Send("دوره مورد نظر برای شروع یافت نشد.", keyboards.MainMenu)
		}
		return SendServiceError(c, fmt.Sprintf("starting/resuming course %d", courseID), err)
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
			return SendServiceError(c, fmt.Sprintf("advancing to next word in course %d", courseID), err)
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

	// ---> ADD THIS BLOCK TO GET THE FULL USER OBJECT <---
	dbUser, err := appServices.User().GetOrCreateUserByTelegramID(c.Sender().ID, c.Sender().Username, c.Sender().FirstName, c.Sender().LastName)
	if err != nil {
		log.Printf("[HandleReturnToMainMenu] Could not get user details for UserID %d to build menu: %v", userID, err)
		// Fallback to sending a default menu if user can't be fetched
		return c.Send("بازگشت به منوی اصلی.", keyboards.MainMenu)
	}
	// ---> END OF BLOCK <---

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
			neutralText = formatters.EscapeMarkdownV2(neutralText)

			editPreviousMessage(c, messageToDelete.ChatID, messageToDelete.MessageID, neutralText, nil)
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
	returnText = formatters.EscapeMarkdownV2(returnText)

	return c.Send(returnText, keyboards.NewMainMenu(dbUser.IsAdmin), telebot.ModeMarkdownV2)
}
