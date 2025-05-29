package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

// findActiveQuizAttemptForBlock remains the same
func findActiveQuizAttemptForBlock(db *gorm.DB, userID uint, courseID uint, progressAtBlockEnd uint) (*models.QuizAttempt, error) {
	var attempt models.QuizAttempt
	err := db.Joins("JOIN quizzes ON quizzes.id = quiz_attempts.quiz_id").
		Where("quiz_attempts.user_id = ? AND quizzes.course_id = ? AND quizzes.trigger_progress = ? AND quiz_attempts.is_completed = ?",
			userID, courseID, progressAtBlockEnd, false).
		// Preload("Quiz.QuizQuesetions.Options"). // Not strictly needed here, will load Quiz in sendCurrentQuizQuestion
		Order("quiz_attempts.created_at DESC").
		First(&attempt).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &attempt, nil
}

// startQuizFlow remains mostly the same
func startQuizFlow(ctx telebot.Context, db *gorm.DB, userID uint, courseID uint, userCourseProgress uint) error {
	progressForQuizBlock := userCourseProgress

	activeAttempt, err := findActiveQuizAttemptForBlock(db, userID, courseID, progressForQuizBlock)
	if err != nil {
		fmt.Printf("Error finding active quiz attempt for user %d, course %d, block %d: %v\n", userID, courseID, progressForQuizBlock, err)
		return ctx.Send("خطایی در بررسی آزمون‌های قبلی شما رخ داد.")
	}

	var currentAttempt *models.QuizAttempt
	if activeAttempt != nil {
		ctx.Send("شما یک آزمون نیمه‌تمام برای این بخش دارید. ادامه می‌دهیم...")
		currentAttempt = activeAttempt
		// Reset message ID if resuming, so it sends a new message first
		// Or, ensure the message ID is still valid and editable.
		// For simplicity, let's assume resuming means sending the question fresh.
		// A more advanced resume could try to find and edit the old message if its ID was stored robustly.
		// For now, we'll clear it to force a new Send, which then gets stored.
		if currentAttempt.CurrentQuestionMessageID != 0 {
			if err := db.Model(&currentAttempt).Update("current_question_message_id", 0).Error; err != nil {
				fmt.Printf("Error clearing message ID for resuming attempt %d: %v\n", currentAttempt.ID, err)
			}
			currentAttempt.CurrentQuestionMessageID = 0
		}

	} else {
		ctx.Send(fmt.Sprintf("🎉 زمان آزمون! شما %d کلمه را مطالعه کرده‌اید. بیایید ببینیم چقدر یاد گرفته‌اید.", userCourseProgress))
		newAttempt, errCreate := createQuizAndAttemptForUser(db, userID, courseID, progressForQuizBlock)
		if errCreate != nil {
			fmt.Printf("Error in startQuizFlow when creating new quiz attempt for user %d, course %d: %v\n", userID, courseID, errCreate)
			return ctx.Send("متاسفانه مشکلی در ایجاد آزمون شما پیش آمد.")
		}
		currentAttempt = newAttempt
	}

	var quizForAttempt models.Quiz
	if err := db.Preload("QuizQuesetions.Options").First(&quizForAttempt, currentAttempt.QuizID).Error; err != nil {
		fmt.Printf("Error fetching quiz (ID: %d) for attempt (ID: %d): %v\n", currentAttempt.QuizID, currentAttempt.ID, err)
		return ctx.Send("خطا در بارگذاری جزئیات آزمون.")
	}
	if quizForAttempt.QuestionCount == 0 {
		return ctx.Send("آزمون ایجاد شد اما هیچ سوالی در آن وجود ندارد. لطفا به ادمین اطلاع دهید.")
	}

	return sendCurrentQuizQuestion(ctx, db, currentAttempt.ID)
}

// sendCurrentQuizQuestion will now try to edit the message
// In internal/bot/handlers/quiz_handler.go

// sendCurrentQuizQuestion will now try to edit the message and use ctx.Bot().Send/Edit
func sendCurrentQuizQuestion(ctx telebot.Context, db *gorm.DB, attemptID uint) error {
	var attempt models.QuizAttempt
	if err := db.First(&attempt, attemptID).Error; err != nil {
		// It's better to log internal errors and send a generic message to the user.
		fmt.Printf("sendCurrentQuizQuestion: Error fetching attempt %d: %v\n", attemptID, err)
		return ctx.Send("خطا در بارگذاری وضعیت آزمون شما.")
	}

	if attempt.IsCompleted {
		// This quiz attempt is done. Finalize might have already run or should run.
		fmt.Printf("sendCurrentQuizQuestion: Attempt %d is already completed.\n", attemptID)
		user, _ := fetchUser(ctx, db) // fetchUser might return an error if ctx is unusual
		return finalizeQuizAttempt(ctx, db, user, attemptID)
	}

	var quiz models.Quiz
	if err := db.Preload("QuizQuesetions.Options").First(&quiz, attempt.QuizID).Error; err != nil {
		fmt.Printf("sendCurrentQuizQuestion: Error fetching quiz %d for attempt %d: %v\n", attempt.QuizID, attemptID, err)
		return ctx.Send("خطا در بارگذاری سوالات آزمون.")
	}

	questionNum := attempt.CurrentQuestionNum
	if questionNum < 0 || questionNum >= int(quiz.QuestionCount) {
		fmt.Printf("sendCurrentQuizQuestion: Invalid questionNum %d for quiz %d (attempt %d). Total questions: %d. Finalizing.\n",
			questionNum, quiz.ID, attemptID, quiz.QuestionCount)
		user, _ := fetchUser(ctx, db)
		return finalizeQuizAttempt(ctx, db, user, attemptID)
	}

	currentQuestion := quiz.QuizQuesetions[questionNum]
	questionText := fmt.Sprintf("سوال %d از %d:\n\n%s", questionNum+1, quiz.QuestionCount, currentQuestion.Text)
	optionsMarkup := keyboards.QuizQuestionOptions(currentQuestion.Options, attemptID, currentQuestion.ID)

	var finalMessage *telebot.Message
	var operationError error

	if attempt.CurrentQuestionMessageID != 0 && ctx.Chat() != nil {
		targetToEdit := telebot.StoredMessage{
			MessageID: strconv.Itoa(attempt.CurrentQuestionMessageID),
			ChatID:    ctx.Chat().ID,
		}
		// Use ctx.Bot().Edit() to get the *telebot.Message object back
		editedMessage, errEdit := ctx.Bot().Edit(targetToEdit, questionText, optionsMarkup)
		if errEdit == nil {
			finalMessage = editedMessage
		} else {
			fmt.Printf("Error editing message ID %s for attempt %d: %v. Will send a new message.\n",
				targetToEdit.MessageID, attemptID, errEdit)
			// If editing failed, we must send a new message.
			// Clear the old message ID from the database attempt so we don't try to edit it again if this function is re-entered.
			if dbErr := db.Model(&models.QuizAttempt{}).Where("id = ?", attempt.ID).Update("current_question_message_id", 0).Error; dbErr != nil {
				fmt.Printf("Failed to clear CurrentQuestionMessageID for attempt %d after edit failure: %v\n", attemptID, dbErr)
				// This is an internal state issue, but we'll still try to send a new message for the user.
			}
			// No need to update attempt.CurrentQuestionMessageID in memory here, the next block handles it.
		}
	}

	// If no message has been successfully set yet (i.e., it's the first question or editing failed)
	if finalMessage == nil {
		// Use ctx.Bot().Send() to get the *telebot.Message object back
		sentMessage, errSend := ctx.Bot().Send(ctx.Chat(), questionText, optionsMarkup)
		if errSend == nil {
			finalMessage = sentMessage
		} else {
			operationError = errSend // Store the sending error
		}
	}

	// If there was any error in the process and we don't have a final message
	if finalMessage == nil {
		if operationError == nil { // Should not happen if finalMessage is nil, indicates a logic flaw
			operationError = errors.New("unknown error in sendCurrentQuizQuestion, no message was sent or edited")
		}
		fmt.Printf("Error in sendCurrentQuizQuestion for attempt %d: %v\n", attemptID, operationError)
		return ctx.Send("مشکلی در نمایش سوال آزمون پیش آمد.")
	}

	// We have a finalMessage (either newly sent or successfully edited).
	// Update the attempt with its ID if it's different from what's stored,
	// or if the stored one was just cleared due to edit failure.
	if attempt.CurrentQuestionMessageID != finalMessage.ID {
		if saveErr := db.Model(&models.QuizAttempt{}).Where("id = ?", attempt.ID).Update("current_question_message_id", finalMessage.ID).Error; saveErr != nil {
			fmt.Printf("Error saving CurrentQuestionMessageID %d for attempt %d: %v\n", finalMessage.ID, attemptID, saveErr)
			// This is not immediately fatal for the user at this stage, but the next edit might fail or send new.
		}
	}

	return nil
}

// handleQuizAnswerCallback will no longer send immediate feedback
func handleQuizAnswerCallback(ctx telebot.Context, db *gorm.DB) error {
	// Acknowledge the callback immediately.
	// No text feedback in the acknowledgment as per requirement.
	defer ctx.Respond(&telebot.CallbackResponse{})

	user, err := fetchUser(ctx, db)
	if err != nil {
		fmt.Printf("Error fetching user in handleQuizAnswerCallback: %v\n", err)
		return nil // Avoid sending error through callback response if user not found
	}

	payload := strings.TrimSpace(ctx.Callback().Data)
	parts := strings.Split(payload, ":")
	if len(parts) != 4 || parts[0] != "quiz_ans" {
		fmt.Printf("Invalid callback payload: %s\n", payload)
		return nil // Silently ignore invalid payloads or log them
	}

	attemptIDFromCb, _ := strconv.ParseUint(parts[1], 10, 64)
	optionIDFromCb, _ := strconv.ParseUint(parts[3], 10, 64)

	var attempt models.QuizAttempt
	if err := db.First(&attempt, uint(attemptIDFromCb)).Error; err != nil {
		fmt.Printf("Error fetching quiz attempt ID %d in callback: %v\n", attemptIDFromCb, err)
		// Don't send message here as ctx.Respond already happened.
		// Button will just stop loading. Consider logging.
		return nil
	}

	if attempt.IsCompleted {
		// Quiz already completed. Maybe send an alert with ctx.Respond if desired.
		// For now, just log and ignore further interaction.
		fmt.Printf("Attempt %d already completed. Callback ignored.\n", attempt.ID)
		return nil
	}
	if attempt.UserID != user.ID {
		fmt.Printf("User mismatch for attempt %d. Expected %d, callback from %d\n", attempt.ID, attempt.UserID, user.ID)
		return nil
	}

	var chosenOption models.QuizQuestionOption
	if err := db.First(&chosenOption, uint(optionIDFromCb)).Error; err != nil {
		fmt.Printf("Error fetching chosen option ID %d: %v\n", optionIDFromCb, err)
		return nil
	}

	// Verify the question being answered is the current one for the attempt
	var quizForAttemptCheck models.Quiz
	if err := db.Preload("QuizQuesetions").First(&quizForAttemptCheck, attempt.QuizID).Error; err != nil {
		fmt.Printf("Error loading quiz %d for attempt %d to verify question: %v\n", attempt.QuizID, attempt.ID, err)
		return nil
	}
	if attempt.CurrentQuestionNum < 0 || attempt.CurrentQuestionNum >= int(quizForAttemptCheck.QuestionCount) {
		fmt.Printf("Attempt %d current question num %d is out of bounds for quiz %d. Finalizing.\n", attempt.ID, attempt.CurrentQuestionNum, quizForAttemptCheck.ID)
		return finalizeQuizAttempt(ctx, db, user, attempt.ID)
	}
	expectedQuestionID := quizForAttemptCheck.QuizQuesetions[attempt.CurrentQuestionNum].ID
	if chosenOption.QuizQuestionID != expectedQuestionID {
		fmt.Printf("Option chosen (for Q_ID %d) does not match current question (Q_ID %d) in attempt %d.\n", chosenOption.QuizQuestionID, expectedQuestionID, attempt.ID)
		// Resend the correct current question because user might have clicked an old button
		// or there's a state mismatch.
		return sendCurrentQuizQuestion(ctx, db, attempt.ID)
	}

	// Record the answer
	quizAnswer := models.QuizAnswer{
		QuizAttemptID:        attempt.ID,
		QuizQuestionID:       chosenOption.QuizQuestionID,
		QuizQuestionOptionID: chosenOption.ID,
		IsCorrect:            chosenOption.IsCorrect,
	}
	if err := db.Create(&quizAnswer).Error; err != nil {
		fmt.Printf("Error saving quiz answer for attempt %d: %v\n", attempt.ID, err)
		return nil // Error saving answer, don't proceed.
	}

	// --- NO IMMEDIATE FEEDBACK ---
	// The message containing the clicked button will be edited by the next call to sendCurrentQuizQuestion
	// or by finalizeQuizAttempt. No need to delete ctx.Callback().Message here.

	// Move to the next question or finalize
	attempt.CurrentQuestionNum++
	if err := db.Model(&attempt).Update("current_question_num", attempt.CurrentQuestionNum).Error; err != nil {
		fmt.Printf("Error updating CurrentQuestionNum for attempt %d: %v\n", attempt.ID, err)
		return nil
	}

	if attempt.CurrentQuestionNum >= int(quizForAttemptCheck.QuestionCount) {
		return finalizeQuizAttempt(ctx, db, user, attempt.ID)
	} else {
		return sendCurrentQuizQuestion(ctx, db, attempt.ID)
	}
}

func finalizeQuizAttempt(ctx telebot.Context, db *gorm.DB, user *models.User, attemptID uint) error {
	var attempt models.QuizAttempt
	// Preload QuizAnswers for score calculation and review.
	if err := db.Preload("QuizAnswers").First(&attempt, attemptID).Error; err != nil {
		fmt.Printf("Error loading quiz attempt %d: %v\n", attemptID, err)
		return ctx.Send("خطا در بارگذاری نتیجه آزمون.")
	}

	// Fetch the associated Quiz with its Questions and their Options for the review.
	var quizInfo models.Quiz
	if err := db.Preload("QuizQuesetions.Options").First(&quizInfo, attempt.QuizID).Error; err != nil {
		fmt.Printf("Error loading quiz info (QuizID: %d) for attempt %d with questions/options: %v\n", attempt.QuizID, attemptID, err)
		return ctx.Send("خطا در دریافت اطلاعات کامل آزمون برای مرور.")
	}

	// Calculate score and mark as completed if not already done.
	if !attempt.IsCompleted {
		correctAnswers := 0
		for _, ans := range attempt.QuizAnswers {
			if ans.IsCorrect {
				correctAnswers++
			}
		}
		attempt.Score = correctAnswers
		attempt.IsCompleted = true
		if err := db.Save(&attempt).Error; err != nil {
			fmt.Printf("Error saving final quiz attempt details for attempt %d: %v\n", attemptID, err)
			// Continue to inform user, but log the error.
		}
	}

	totalQuestions := int(quizInfo.QuestionCount)
	if totalQuestions == 0 {
		totalQuestions = WordsPerQuizBlock // Fallback, ensure WordsPerQuizBlock is defined
		fmt.Printf("Warning: quizInfo.QuestionCount was 0 for quiz %d (attempt %d). Using default %d.\n", quizInfo.ID, attemptID, totalQuestions)
	}

	// Initial result message
	resultMessageText := fmt.Sprintf("🏁 آزمون تمام شد!\n\nشما به %d سوال از %d سوال پاسخ صحیح دادید.", attempt.Score, totalQuestions)

	// --- Begin Detailed Review Section ---
	detailedReview := "\n\n📝 مرور سوالات:\n"

	// Create a map for quick lookup of user's answers
	userAnswersMap := make(map[uint]models.QuizAnswer)
	var allChosenOptionIDs []uint
	for _, ans := range attempt.QuizAnswers {
		userAnswersMap[ans.QuizQuestionID] = ans
		allChosenOptionIDs = append(allChosenOptionIDs, ans.QuizQuestionOptionID)
	}

	// Fetch texts of all options chosen by the user in one query
	chosenOptionsTextMap := make(map[uint]string)
	if len(allChosenOptionIDs) > 0 {
		var fetchedChosenOptions []models.QuizQuestionOption
		if err := db.Where("id IN ?", allChosenOptionIDs).Find(&fetchedChosenOptions).Error; err == nil {
			for _, opt := range fetchedChosenOptions {
				chosenOptionsTextMap[opt.ID] = opt.Text
			}
		} else {
			fmt.Printf("Error fetching chosen option texts for review (AttemptID: %d): %v\n", attemptID, err)
		}
	}

	// Loop through questions in the order they appear in quizInfo.QuizQuesetions
	// (GORM usually orders by primary key if not specified, ensure this order is intended for review)
	for i, question := range quizInfo.QuizQuesetions {
		detailedReview += fmt.Sprintf("\n%d. سوال: %s\n", i+1, question.Text)

		userChosenText := "پاسخ نداده"
		userCorrectStatus := "❌" // Default to incorrect or unanswered

		if userAnswer, found := userAnswersMap[question.ID]; found {
			if text, ok := chosenOptionsTextMap[userAnswer.QuizQuestionOptionID]; ok {
				userChosenText = text
			} else {
				userChosenText = "(خطا در بازیابی متن پاسخ شما)"
			}
			if userAnswer.IsCorrect {
				userCorrectStatus = "✅"
			}
		}
		detailedReview += fmt.Sprintf("   شما پاسخ دادید: %s (%s)\n", userChosenText, userCorrectStatus)

		// Find and list correct option(s) for this question
		var correctOptionTexts []string
		for _, opt := range question.Options {
			if opt.IsCorrect {
				correctOptionTexts = append(correctOptionTexts, opt.Text)
			}
		}
		if len(correctOptionTexts) > 0 {
			detailedReview += fmt.Sprintf("   پاسخ صحیح: %s\n", strings.Join(correctOptionTexts, " / "))
		} else {
			detailedReview += "   (پاسخ صحیحی برای این سوال ثبت نشده است)\n"
		}
	}
	resultMessageText += detailedReview
	// --- End Detailed Review Section ---

	// Edit the last question's message or send a new one
	if attempt.CurrentQuestionMessageID != 0 && ctx.Chat() != nil {
		targetToEdit := telebot.StoredMessage{
			MessageID: strconv.Itoa(attempt.CurrentQuestionMessageID),
			ChatID:    ctx.Chat().ID,
		}
		_, editErr := ctx.Bot().Edit(targetToEdit, resultMessageText, &telebot.ReplyMarkup{}) // Empty markup removes buttons
		if editErr != nil {
			fmt.Printf("Error editing final quiz message %d for attempt %d: %v. Sending as new message.\n", attempt.CurrentQuestionMessageID, attemptID, editErr)
			ctx.Send(resultMessageText)
		}
	} else {
		ctx.Send(resultMessageText)
	}

	// Clear the message ID now that the quiz is done with this message
	if attempt.CurrentQuestionMessageID != 0 {
		if err := db.Model(&models.QuizAttempt{}).Where("id = ?", attempt.ID).Update("current_question_message_id", 0).Error; err != nil {
			fmt.Printf("Error clearing CurrentQuestionMessageID for completed attempt %d: %v\n", attemptID, err)
		}
	}

	// Fetch user if nil (for UserCourse update)
	if user == nil && attempt.UserID != 0 {
		var tempUser models.User
		if err := db.First(&tempUser, attempt.UserID).Error; err == nil {
			user = &tempUser
		} else {
			fmt.Printf("FinalizeQuizAttempt: Could not fetch user %d for UserCourse update.\n", attempt.UserID)
			// The review has been sent, but course progress update might fail.
			// Send a more generic error or just log.
			return ctx.Send("نتیجه آزمون نمایش داده شد، اما خطایی در به‌روزرسانی پیشرفت دوره رخ داد.")
		}
	}
	if user == nil {
		fmt.Printf("FinalizeQuizAttempt: User object is nil for attempt %d, cannot update course progress.\n", attemptID)
		return ctx.Send("خطای داخلی: اطلاعات کاربر برای به‌روزرسانی دوره یافت نشد.")
	}

	// Update UserCourse progress
	var uc models.UserCourse
	if err := db.Where("user_id = ? AND course_id = ?", user.ID, quizInfo.CourseID).First(&uc).Error; err != nil {
		return ctx.Send("خطا در یافتن اطلاعات دوره شما برای به‌روزرسانی پیشرفت.")
	}

	passThreshold := QuizPassThresholdCorrectAnswers // Ensure this constant is defined
	wordsInBlock := WordsPerQuizBlock                // Ensure this constant is defined

	if attempt.Score >= passThreshold {
		ctx.Send(fmt.Sprintf("🎉 تبریک! شما آزمون را با موفقیت گذراندید و می‌توانید به یادگیری ادامه دهید."), keyboards.Course())
	} else {
		ctx.Send(fmt.Sprintf("😔 متاسفانه حد نصاب قبولی (%d پاسخ صحیح) را کسب نکردید.", passThreshold))

		triggerProgressForThisQuiz := quizInfo.TriggerProgress
		if triggerProgressForThisQuiz == 0 {
			fmt.Printf("Warning: quizInfo.TriggerProgress was 0 for quiz %d. Progress reset might be inaccurate.\n", quizInfo.ID)
		}

		startOfBlockProgress := triggerProgressForThisQuiz - uint(wordsInBlock) + 1
		if triggerProgressForThisQuiz < uint(wordsInBlock) { // Quiz was for the first block
			startOfBlockProgress = 1
		}
		if startOfBlockProgress < 1 {
			startOfBlockProgress = 1
		}

		if uc.Progress >= startOfBlockProgress { // Only reset if relevant
			uc.Progress = startOfBlockProgress
		}

		if err := db.Save(&uc).Error; err != nil {
			return ctx.Send("خطا در به‌روزرسانی پیشرفت شما پس از آزمون.")
		}
		ctx.Send(fmt.Sprintf("پیشرفت شما به کلمه شماره %d بازگردانده شد. لطفا دوباره کلمات را مطالعه کنید.", uc.Progress), keyboards.Course())
	}
	return nil
}
