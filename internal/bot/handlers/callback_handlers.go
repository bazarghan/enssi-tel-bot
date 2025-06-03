package handlers

import (
	"log"
	"strconv"
	"strings"

	"errors" // For errors.Is
	"github.com/2000ostd/enssi-tel-bot/internal/bot/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/services"
	"github.com/2000ostd/enssi-tel-bot/internal/services/quiz" // For quiz.ErrAttemptNotFound etc.
	"gopkg.in/telebot.v4"
)

// HandleQuizAnswerCallback processes answers submitted via inline quiz buttons.
func HandleQuizAnswerCallback(c telebot.Context, appServices *services.AppServices) error {
	cb := c.Callback()
	if cb == nil {
		log.Println("[HandleQuizAnswerCallback] Received nil callback.")
		return nil // Should not happen
	}

	// Acknowledge callback immediately to stop the loading animation on the button.
	defer func() {
		if err := c.Respond(); err != nil {
			log.Printf("[HandleQuizAnswerCallback] Error responding to callback: %v", err)
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
		// Cannot easily send a message here as callback might have been responded to.
		return nil
	}

	payload := strings.TrimSpace(cb.Data)
	parts := strings.Split(payload, ":")

	if len(parts) != 3 || parts[0] != strings.TrimSuffix(keyboards.QuizAnswerCallbackPrefix, ":") { // "qa"
		log.Printf("[HandleQuizAnswerCallback] UserID %d: Invalid callback payload format: %s", dbUser.ID, payload)
		return nil
	}

	attemptIDVal, errAttempt := strconv.ParseUint(parts[1], 10, 64)
	optionIDVal, errOption := strconv.ParseUint(parts[2], 10, 64)

	if errAttempt != nil || errOption != nil || attemptIDVal == 0 || optionIDVal == 0 {
		log.Printf("[HandleQuizAnswerCallback] UserID %d: Invalid IDs in callback payload: %s. AttemptErr: %v, OptionErr: %v", dbUser.ID, payload, errAttempt, errOption)
		return nil
	}
	attemptID := uint(attemptIDVal)
	optionID := uint(optionIDVal)

	log.Printf("[HandleQuizAnswerCallback] UserID %d submitting answer for AttemptID %d, OptionID %d", dbUser.ID, attemptID, optionID)

	submissionResult, err := appServices.Quiz().SubmitAnswer(attemptID, optionID, dbUser.ID)
	if err != nil {
		log.Printf("[HandleQuizAnswerCallback] UserID %d, AttemptID %d: Error submitting answer: %v", dbUser.ID, attemptID, err)
		// Try to edit the message to show an error
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
			_, editErr := c.Bot().Edit(cb.Message, formatters.EscapeMarkdownV2(errorText), &telebot.ReplyMarkup{}) // Remove buttons
			if editErr != nil {
				log.Printf("[HandleQuizAnswerCallback] Failed to edit message with error for AttemptID %d: %v", attemptID, editErr)
			}
		}
		return nil // Error handled by logging and potentially editing message
	}

	// --- Process Submission Result ---
	originalQuestionMessageID := ""
	if submissionResult.CurrentQuestionMessageID != 0 {
		originalQuestionMessageID = strconv.Itoa(submissionResult.CurrentQuestionMessageID)
	} else if cb.Message != nil { // Fallback
		originalQuestionMessageID = strconv.Itoa(cb.Message.ID)
	}

	if submissionResult.IsQuizNowCompleted {
		finalResult := submissionResult.FinalQuizResult
		log.Printf("[HandleQuizAnswerCallback] UserID %d: Quiz AttemptID %d completed. Score: %d/%d. Passed: %t",
			dbUser.ID, finalResult.AttemptID, finalResult.Score, finalResult.TotalQuestions, finalResult.Passed)

		// Edit the last question's message to show the full result, removing buttons.
		if originalQuestionMessageID != "" && cb.Message != nil { // cb.Message provides ChatID
			fullResultMessage := formatters.FormatQuizResultForDisplay(finalResult)
			editPreviousMessage(c, cb.Message.Chat.ID, originalQuestionMessageID, fullResultMessage, nil) // nil markup removes buttons
		} else {
			// If we can't edit, send as new messages (less ideal UX)
			if err := c.Send(formatters.EscapeMarkdownV2(finalResult.ResultMessage), telebot.ModeMarkdownV2); err != nil {
				log.Printf("Error sending quiz result message for attempt %d: %v", finalResult.AttemptID, err)
			}
			if finalResult.ReviewText != "" {
				if err := c.Send(formatters.EscapeMarkdownV2(finalResult.ReviewText), telebot.ModeMarkdownV2); err != nil {
					log.Printf("Error sending quiz review text for attempt %d: %v", finalResult.AttemptID, err)
				}
			}
		}

		// After showing quiz results, determine next step in the course using CourseService.HandleQuizCompletion
		// This service call itself will return a new LearningContext.
		nextLc, errHc := appServices.Course().HandleQuizCompletion(dbUser.ID, finalResult.CourseID, finalResult)
		if errHc != nil {
			log.Printf("[HandleQuizAnswerCallback] UserID %d, CourseID %d: Error in CourseService.HandleQuizCompletion: %v", dbUser.ID, finalResult.CourseID, errHc)
			return sendServiceError(c, "handling quiz completion", errHc)
		}
		return sendLearningContext(c, nextLc, appServices) // This will update menu state and send next content

	} else if submissionResult.NextQuestionState != nil {
		nextState := submissionResult.NextQuestionState
		log.Printf("[HandleQuizAnswerCallback] UserID %d: Quiz AttemptID %d continues. Next question #%d (%s)",
			dbUser.ID, nextState.AttemptID, nextState.CurrentQuestionNum, nextState.QuestionText)

		// Edit the previous message to show the new question.
		if originalQuestionMessageID != "" && cb.Message != nil {
			//questionText := fmt.Sprintf("سوال %d از %d:\n%s", nextState.CurrentQuestionNum+1, nextState.TotalQuestionsInQuiz, formatters.EscapeMarkdownV2(nextState.QuestionText))
			questionText := nextState.QuestionText
			optionsMarkup := keyboards.QuizQuestionOptionsKeyboard(nextState.Options, nextState.AttemptID)

			editable := &telebot.Message{ID: cb.Message.ID, Chat: cb.Message.Chat}
			if originalQuestionMessageIDFromService, errConv := strconv.Atoi(originalQuestionMessageID); errConv == nil {
				editable.ID = originalQuestionMessageIDFromService
			}

			sentMsg, editErr := c.Bot().Edit(editable, questionText, optionsMarkup, telebot.ModeMarkdownV2)
			if editErr != nil {
				log.Printf("[HandleQuizAnswerCallback] Failed to edit quiz message (MsgID: %s) for next question. Error: %v. Sending new.", originalQuestionMessageID, editErr)
				// Fallback: send as a new message
				return sendQuizQuestion(c, nextState, appServices.Quiz())
			}
			// If edit successful, update CurrentQuestionMessageID for the attempt with the new (same) message ID
			if sentMsg != nil { // sentMsg is the edited message
				if errUpdateID := appServices.Quiz().UpdateQuizAttemptMessageID(nextState.AttemptID, sentMsg.ID); errUpdateID != nil {
					log.Printf("[HandleQuizAnswerCallback] UserID %d, AttemptID %d: Failed to update message ID after editing for next question: %v", dbUser.ID, nextState.AttemptID, errUpdateID)
				}
			}
		} else {
			// No original message context to edit, send as new.
			return sendQuizQuestion(c, nextState, appServices.Quiz())
		}
	} else {
		log.Printf("[HandleQuizAnswerCallback] UserID %d, AttemptID %d: Submission result inconclusive.", dbUser.ID, attemptID)
		if cb.Message != nil {
			_, editErr := c.Bot().Edit(cb.Message, "پاسخ شما پردازش شد، اما وضعیت بعدی نامشخص است.", &telebot.ReplyMarkup{})
			if editErr != nil {
				log.Printf("[HandleQuizAnswerCallback] Failed to edit message for inconclusive result: %v", editErr)
			}
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
	defer c.Respond()

	dbUser, err := appServices.User().GetOrCreateUserByTelegramID(c.Sender().ID, c.Sender().Username, c.Sender().FirstName, c.Sender().LastName)
	if err != nil {
		return sendServiceError(c, "identifying user for course selection", err)
	}

	payload := strings.TrimSpace(cb.Data) // Format: "cs:<course_id>"
	parts := strings.Split(payload, ":")

	if len(parts) != 2 || parts[0] != strings.TrimSuffix(keyboards.CourseDetailsCallbackPrefix, ":") { // "cs"
		log.Printf("[HandleCourseSelectionCallback] UserID %d: Invalid callback payload: %s", dbUser.ID, payload)
		return nil
	}

	courseIDVal, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil || courseIDVal == 0 {
		log.Printf("[HandleCourseSelectionCallback] UserID %d: Invalid course ID in payload: %s. Error: %v", dbUser.ID, payload, err)
		return nil
	}
	courseID := uint(courseIDVal)

	// Remove the inline keyboard message
	if cb.Message != nil {
		c.Bot().Delete(cb.Message)
	}

	return displayCourseOverview(c, dbUser.ID, courseID, appServices)
}
