package handlers

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time" // For MarkReviewSessionCompleted

	"github.com/2000ostd/enssi-tel-bot/internal/bot/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/models" // For models.QuizType
	"github.com/2000ostd/enssi-tel-bot/internal/services"
	"github.com/2000ostd/enssi-tel-bot/internal/services/quiz" // For quiz.ErrAttemptNotFound etc.
	"gopkg.in/telebot.v4"
)

// HandleQuizAnswerCallback processes answers submitted via inline quiz buttons.
func HandleQuizAnswerCallback(c telebot.Context, appServices *services.AppServices) error {
	cb := c.Callback()
	if cb == nil {
		log.Println("[HandleQuizAnswerCallback] Received nil callback.")
		return nil
	}

	// Acknowledge callback immediately
	defer func() {
		if err := c.Respond(); err != nil {
			// Log if responding failed, but don't let it mask other errors.
			// It might fail if already responded to (e.g. by sendServiceError in a callback context)
			log.Printf("[HandleQuizAnswerCallback] Info: Error responding to callback (possibly already responded): %v", err)
		}
	}()

	dbUser, err := appServices.User().GetOrCreateUserByTelegramID(
		c.Sender().ID,
		c.Sender().Username,
		c.Sender().FirstName,
		c.Sender().LastName,
	)
	if err != nil {
		log.Printf("[HandleQuizAnswerCallback] Error fetching user (TelegramID %d): %v", c.Sender().ID, err)
		// Attempt to edit message with error if possible
		if cb.Message != nil {
			editPreviousMessage(c, cb.Message.Chat.ID, strconv.Itoa(cb.Message.ID), "خطا در پردازش کاربر.", nil)
		}
		return nil // Error handled
	}

	// NOTE: No CheckAndInitiateReview here because user is already IN a quiz.
	// A review session wouldn't interrupt an ongoing quiz.

	payload := strings.TrimSpace(cb.Data)
	parts := strings.Split(payload, ":")

	// Expecting "quiz_ans:<attempt_id>:<option_id>"
	if len(parts) != 3 || parts[0] != strings.TrimSuffix(keyboards.QuizAnswerCallbackPrefix, ":") {
		log.Printf("[HandleQuizAnswerCallback] UserID %d: Invalid callback payload format: %s", dbUser.ID, payload)
		if cb.Message != nil {
			editPreviousMessage(c, cb.Message.Chat.ID, strconv.Itoa(cb.Message.ID), "خطای دکمه.", nil)
		}
		return nil
	}

	attemptIDVal, errAttempt := strconv.ParseUint(parts[1], 10, 64)
	optionIDVal, errOption := strconv.ParseUint(parts[2], 10, 64)

	if errAttempt != nil || errOption != nil || attemptIDVal == 0 { // optionIDVal can be 0 if it's a special non-answer button
		log.Printf("[HandleQuizAnswerCallback] UserID %d: Invalid IDs in callback payload: %s. AttemptErr: %v, OptionErr: %v", dbUser.ID, payload, errAttempt, errOption)
		if cb.Message != nil {
			editPreviousMessage(c, cb.Message.Chat.ID, strconv.Itoa(cb.Message.ID), "خطای داخلی دکمه.", nil)
		}
		return nil
	}
	attemptID := uint(attemptIDVal)
	optionID := uint(optionIDVal)

	log.Printf("[HandleQuizAnswerCallback] UserID %d submitting answer for AttemptID %d, OptionID %d", dbUser.ID, attemptID, optionID)

	submissionResult, err := appServices.Quiz().SubmitAnswer(attemptID, optionID, dbUser.ID)
	if err != nil {
		log.Printf("[HandleQuizAnswerCallback] UserID %d, AttemptID %d: Error submitting answer: %v", dbUser.ID, attemptID, err)

		// ---> MODIFICATION IS HERE <---
		// Handle the new error type for duplicate clicks
		if errors.Is(err, quiz.ErrQuestionAlreadyAnswered) {
			log.Printf("[HandleQuizAnswerCallback] Ignored duplicate answer for AttemptID %d. No action taken.", attemptID)
			// Return nil immediately. The 'defer c.Respond()' at the top of the function
			// will acknowledge the callback to stop the loading animation on the button,
			// but we won't send any message or edit the existing one.
			return nil
		}
		// ---> END OF MODIFICATION <---
		var errorText string
		switch {
		case errors.Is(err, quiz.ErrAttemptNotFound):
			errorText = "این آزمون دیگر فعال نیست یا منقضی شده است."
		case errors.Is(err, quiz.ErrAttemptAlreadyCompleted):
			errorText = "شما قبلا به این آزمون پاسخ داده‌اید."
		case errors.Is(err, quiz.ErrInvalidOption):
			errorText = "گزینه انتخابی نامعتبر است."
		case errors.Is(err, quiz.ErrUserMismatch):
			errorText = "شما مجاز به پاسخ به این آزمون نیستید."
		default:
			errorText = "مشکلی در ثبت پاسخ شما بوجود آمد."
		}
		if cb.Message != nil {
			// Edit the original quiz message to show the error and remove buttons
			editPreviousMessage(c, cb.Message.Chat.ID, strconv.Itoa(cb.Message.ID), formatters.EscapeMarkdownV2(errorText), nil)
		}
		return nil // Error handled
	}

	// --- Process Submission Result ---
	originalQuestionMessageID := submissionResult.CurrentQuestionMessageID // Use the one from submission result

	if submissionResult.IsQuizNowCompleted {
		finalResult := submissionResult.FinalQuizResult
		log.Printf("[HandleQuizAnswerCallback] UserID %d: Quiz AttemptID %d (Type: %s) completed. Score: %d/%d. Passed: %t",
			dbUser.ID, finalResult.AttemptID, finalResult.QuizType, finalResult.Score, finalResult.TotalQuestions, finalResult.Passed)

		// Edit the last question's message to show the full result, removing buttons.
		if originalQuestionMessageID != 0 && cb.Message != nil {
			fullResultMessage := formatters.FormatQuizResultForDisplay(finalResult) // This should handle different quiz types
			editPreviousMessage(c, cb.Message.Chat.ID, strconv.Itoa(originalQuestionMessageID), fullResultMessage, nil)
		} else if cb.Message != nil { // Fallback if originalQuestionMessageID was 0 but we have current message
			fullResultMessage := formatters.FormatQuizResultForDisplay(finalResult)
			editPreviousMessage(c, cb.Message.Chat.ID, strconv.Itoa(cb.Message.ID), fullResultMessage, nil)
		} else {
			// If we can't edit, send as new messages (less ideal UX)
			c.Send(formatters.EscapeMarkdownV2(finalResult.ResultMessage), telebot.ModeMarkdownV2)
			if finalResult.ReviewText != "" { // Only send review text if it's populated (e.g. for course block quizzes)
				c.Send(formatters.EscapeMarkdownV2(finalResult.ReviewText), telebot.ModeMarkdownV2)
			}
		}

		// --- Handle post-quiz actions based on quiz type ---
		if finalResult.QuizType == models.QuizTypeReview {
			log.Printf("[HandleQuizAnswerCallback] UserID %d: REVIEW quiz completed.", dbUser.ID)
			if err := appServices.User().MarkReviewSessionCompleted(dbUser.ID, time.Now()); err != nil {
				log.Printf("[HandleQuizAnswerCallback] UserID %d: Error marking review session completed: %v", dbUser.ID, err)
			}
			appServices.User().UpdateUserLastMenu(dbUser.ID, StateMain)  // Or StateCourseList
			c.Send("جلسه مرور شما به پایان رسید! 👍", keyboards.MainMenu) // Send with main menu
			return nil                                                   // End of flow for review quiz
		}

		// For COURSE_BLOCK quizzes:
		if finalResult.QuizType == models.QuizTypeCourseBlock {
			nextLc, errHc := appServices.Course().HandleQuizCompletion(dbUser.ID, finalResult.CourseID, finalResult)
			if errHc != nil {
				log.Printf("[HandleQuizAnswerCallback] UserID %d, CourseID %d: Error in CourseService.HandleQuizCompletion: %v", dbUser.ID, finalResult.CourseID, errHc)
				return SendServiceError(c, "handling course quiz completion", errHc)
			}
			return sendLearningContext(c, nextLc, appServices)
		}

		// Fallback for unknown quiz type completion
		log.Printf("[HandleQuizAnswerCallback] UserID %d: Unknown quiz type '%s' completed. Returning to main menu.", dbUser.ID, finalResult.QuizType)
		appServices.User().UpdateUserLastMenu(dbUser.ID, StateMain)
		c.Send("آزمون تکمیل شد.", keyboards.MainMenu)
		return nil

	} else if submissionResult.NextQuestionState != nil { // Quiz continues
		nextState := submissionResult.NextQuestionState
		log.Printf(
			"[HandleQuizAnswerCallback] UserID %d: Quiz AttemptID %d (Type: %s) continues. Next question #%d (%s)",
			dbUser.ID,
			nextState.AttemptID,
			nextState.QuizType,
			nextState.CurrentQuestionNum,
			nextState.QuestionText,
		)

		// Update user's last menu to reflect they are in this specific quiz type
		var currentQuizStateConstant string
		if nextState.QuizType == models.QuizTypeReview {
			currentQuizStateConstant = InReviewQuizMenuState(nextState.AttemptID)
		} else {
			currentQuizStateConstant = InQuizMenuState(nextState.AttemptID)
		}
		appServices.User().UpdateUserLastMenu(dbUser.ID, currentQuizStateConstant)

		//here is the place where questions made

		var textBuilder strings.Builder
		textBuilder.WriteString(nextState.QuestionText)
		textBuilder.WriteString("\n\n")

		for i, opt := range nextState.Options {
			textBuilder.WriteString(fmt.Sprintf("%d\\. %s\n", i+1, formatters.EscapeMarkdownV2(opt.Text)))
		}
		questionText := textBuilder.String()

		optionsMarkup := keyboards.QuizQuestionOptionsKeyboard(nextState.Options, nextState.AttemptID)

		// Edit the previous message to show the new question.
		var editable telebot.Editable
		if originalQuestionMessageID != 0 && cb.Message != nil { // cb.Message gives ChatID
			editable = &telebot.Message{ID: originalQuestionMessageID, Chat: cb.Message.Chat}
		} else if cb.Message != nil { // Fallback to current message if original ID somehow lost
			editable = cb.Message
			log.Printf("[HandleQuizAnswerCallback] Warning: originalQuestionMessageID was 0 for AttemptID %d. Using current callback message for edit.", attemptID)
		} else {
			log.Printf("[HandleQuizAnswerCallback] Error: Cannot edit message for AttemptID %d, no message context.", attemptID)
			// Fallback: send as a new message if no context to edit
			return sendQuizQuestion(c, nextState, appServices.Quiz())
		}

		_, editErr := c.Bot().Edit(editable, questionText, optionsMarkup, telebot.ModeMarkdownV2)
		if editErr != nil {
			log.Printf(
				"[HandleQuizAnswerCallback] Failed to edit quiz message (MsgID: %w) for next question. Error: %v. Sending new.",
				func() string { msgId, _ := editable.MessageSig(); return msgId }(),
				editErr,
			)
			return sendQuizQuestion(c, nextState, appServices.Quiz())
		}

		messageIDToUpdate, _ := strconv.Atoi(func() string { msgId, _ := editable.MessageSig(); return msgId }()) // MessageSig() returns the ID as a string

		if errUpdateID := appServices.Quiz().UpdateQuizAttemptMessageID(nextState.AttemptID, messageIDToUpdate); errUpdateID != nil {
			log.Printf("[HandleQuizAnswerCallback] UserID %d, AttemptID %d: Failed to update message ID after editing for next question: %v", dbUser.ID, nextState.AttemptID, errUpdateID)
		}

	} else {
		log.Printf("[HandleQuizAnswerCallback] UserID %d, AttemptID %d: Submission result inconclusive.", dbUser.ID, attemptID)
		if cb.Message != nil {
			editPreviousMessage(c, cb.Message.Chat.ID, strconv.Itoa(cb.Message.ID), "پاسخ شما پردازش شد، اما وضعیت بعدی نامشخص است.", nil)
		}
	}
	return nil
}

// HandleCourseSelectionCallback processes selection of a course from an inline keyboard.
func HandleCourseSelectionCallback(c telebot.Context, appServices *services.AppServices) error {
	cb := c.Callback()
	if cb == nil {
		return nil
	}
	defer c.Respond() // Acknowledge callback

	dbUser, err := appServices.User().GetOrCreateUserByTelegramID(c.Sender().ID, c.Sender().Username, c.Sender().FirstName, c.Sender().LastName)
	if err != nil {
		return SendServiceError(c, "identifying user for course selection", err)
	}

	// Check for mandatory daily review
	reviewHandled, reviewErr := CheckAndInitiateReview(c, appServices, dbUser)
	if reviewErr != nil {
		return SendServiceError(c, "checking for daily review (course selection)", reviewErr)
	}
	if reviewHandled {
		// If review was handled, it might have sent messages.
		// We should still delete the original course list message if it's an inline keyboard.
		if cb.Message != nil {
			c.Bot().Delete(cb.Message) // Clean up the inline keyboard message
		}
		return nil // Review process has taken over
	}

	payload := strings.TrimSpace(cb.Data)
	// Assuming callback data for course selection is "course_details:<course_id>"
	// as per keyboards.CourseDetailsCallbackPrefix in constants.go
	// The old keyboards.CourseSelectCallbackPrefix ("cs:") might be deprecated.
	expectedPrefix := strings.TrimSuffix(keyboards.CourseDetailsCallbackPrefix, ":") // "course_details"

	parts := strings.Split(payload, ":")
	if len(parts) != 2 || parts[0] != expectedPrefix {
		log.Printf("[HandleCourseSelectionCallback] UserID %d: Invalid callback payload: %s. Expected prefix: '%s'", dbUser.ID, payload, expectedPrefix)
		return nil // Or send an error message
	}

	courseIDVal, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil || courseIDVal == 0 {
		log.Printf("[HandleCourseSelectionCallback] UserID %d: Invalid course ID in payload: %s. Error: %v", dbUser.ID, payload, err)
		return nil
	}
	courseID := uint(courseIDVal)

	// Remove the inline keyboard message (e.g., the list of courses)
	if cb.Message != nil {
		if errDel := c.Bot().Delete(cb.Message); errDel != nil {
			log.Printf("[HandleCourseSelectionCallback] UserID %d: Failed to delete course list message: %v", dbUser.ID, errDel)
		}
	}

	return displayCourseOverview(c, dbUser.ID, courseID, appServices)
}
