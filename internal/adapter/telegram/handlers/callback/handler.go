package callback

import (
	"context"
	"errors"
	"fmt"
	"github.com/bits-and-blooms/bitset"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/achievement"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"

	courseCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/course"
	submitCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/quiz"

	courseQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/course"

	"github.com/2000ostd/enssi-tel-bot/pkg/tgmarkdown"
	"gopkg.in/telebot.v4"
)

const (
	QuizAnswerCallbackPrefix      = "quiz_ans:"
	ShowAchievementCallbackPrefix = "ach_show:"
)

// Handler holds dependencies for all callback handlers.
type Handler struct {
	submitAnswer         submitCmd.SubmitAnswerHandler
	handleQuizCompletion courseCmd.HandleQuizCompletionHandler
	quizRepo             quiz.Repository
	achRepo              achievement.Repository
	imgSvc               achievement.ImageGenerator
	userRepo             user.Repository
	getOverview          courseQueries.GetOverviewHandler
}

// NewHandler creates a new callback handler with all its dependencies.
func NewHandler(
	submitAnswer submitCmd.SubmitAnswerHandler,
	handleQuizCompletion courseCmd.HandleQuizCompletionHandler,
	quizRepo quiz.Repository,
	achRepo achievement.Repository,
	imgSvc achievement.ImageGenerator,
	userRepo user.Repository,
	getOverview courseQueries.GetOverviewHandler,
) *Handler {
	return &Handler{
		submitAnswer:         submitAnswer,
		handleQuizCompletion: handleQuizCompletion,
		quizRepo:             quizRepo,
		achRepo:              achRepo,
		imgSvc:               imgSvc,
		userRepo:             userRepo,
		getOverview:          getOverview,
	}
}

// Handle routes incoming callback queries to the appropriate sub-handler.
func (h *Handler) Handle(c telebot.Context) error {
	cb := c.Callback()
	if cb == nil {
		return nil
	}

	data := strings.TrimSpace(cb.Data)

	if strings.HasPrefix(data, QuizAnswerCallbackPrefix) {
		return h.handleQuizAnswer(c)
	}

	if strings.HasPrefix(data, ShowAchievementCallbackPrefix) {
		return h.handleShowAchievement(c)
	}

	log.Printf("[CallbackHandler] Unhandled callback data: %s", cb.Data)
	return c.Respond()
}

// handleQuizAnswer processes a user's answer to a quiz question.
func (h *Handler) handleQuizAnswer(c telebot.Context) error {
	defer c.Respond()

	ctxUser, ok := c.Get("dbUser").(user.User)
	if !ok {
		return c.Send("Error identifying user.")
	}

	// The telegram callbacks often have whitespace
	data := strings.TrimSpace(c.Callback().Data)

	payload := strings.TrimPrefix(data, QuizAnswerCallbackPrefix)
	parts := strings.Split(payload, ":")
	if len(parts) != 2 {
		log.Printf("Invalid quiz answer payload: %s", payload)
		return nil
	}

	attemptID, _ := strconv.ParseUint(parts[0], 10, 64)
	optionID, _ := strconv.ParseUint(parts[1], 10, 64)

	cmd := submitCmd.SubmitAnswerCommand{
		AttemptID: uint(attemptID),
		OptionID:  uint(optionID),
		UserID:    ctxUser.ID,
	}

	res, err := h.submitAnswer.Handle(context.Background(), cmd)
	if err != nil {
		if errors.Is(err, quiz.ErrQuestionAlreadyAnswered) {
			log.Printf("Error %w, %d: %v", quiz.ErrQuestionAlreadyAnswered, attemptID, err)
			return nil // Silently ignore duplicate clicks
		}
		log.Printf("Error submitting quiz answer for attempt %d: %v", attemptID, err)
		_, errEdit := c.Bot().Edit(c.Callback().Message, "An error occurred while processing your answer.")
		return errEdit
	}

	fullAttempt, err := h.quizRepo.GetAttempt(context.Background(), uint(attemptID))
	if err != nil {
		log.Printf("Could not fetch full attempt %d for formatting: %v", attemptID, err)
	}
	if res.IsCompleted {

		// --- NEW: Check if the completed quiz was a Daily Review ---
		if fullAttempt.Type == quiz.Review {
			// 1. Mark the user's daily review as completed for today.
			if err := h.userRepo.UpdateLastReviewSession(context.Background(), ctxUser.ID); err != nil {
				log.Printf("Failed to update last review session for user %d: %v", ctxUser.ID, err)
			}

			// 2. Format and show the results, just like a normal quiz.
			resultMsg := formatters.FormatQuizResult(res.FinalResult, fullAttempt)
			if _, err := c.Bot().Edit(c.Callback().Message, resultMsg, telebot.ModeMarkdownV2); err != nil {
				log.Printf("Could not edit review quiz result message: %v", err)
			}

			// 3.Build the main menu, explicitly passing `false` for hasPendingReview.
			//    We know the review is complete, so the button should be hidden.
			mainMenuKeyboard := keyboards.NewMainMenu(ctxUser.IsAdmin, false)

			// 4. Send the final confirmation message with the new, clean main menu.
			if _, err := c.Bot().Send(c.Chat(), "آزمون مرور شما به پایان رسید!", mainMenuKeyboard); err != nil {
				log.Printf("Failed to send final review completion message: %v", err)
			}

			// 5.Update the user's state back to 'main' so they are no longer "in_quiz".
			if err := h.userRepo.UpdateLastMenu(context.Background(), ctxUser.ID, "main"); err != nil {
				log.Printf("Failed to update user menu state after review quiz: %v", err)
			}

			return nil
		}
		// --- END OF REVIEW QUIZ LOGIC ---

		completionCmd := courseCmd.HandleQuizCompletionCommand{
			UserID:   ctxUser.ID,
			CourseID: fullAttempt.CourseID,
			Result:   res.FinalResult,
		}
		completionResult, err := h.handleQuizCompletion.Handle(context.Background(), completionCmd)
		if err != nil {
			log.Printf("Error handling quiz completion for attempt %d: %v", attemptID, err)
			_, err = c.Bot().Edit(c.Callback().Message, "Error processing quiz result.")
			return err
		}
		// --- NEW LOGIC: Format and show the quiz result ---

		// 1. Format the result message using your existing formatter.
		resultMsg := formatters.FormatQuizResult(res.FinalResult, fullAttempt)

		// 3. Edit the original quiz message to show the results.
		//    By NOT providing a keyboard here, the old inline buttons are automatically removed.
		if _, err := c.Bot().Edit(c.Callback().Message, resultMsg, telebot.ModeMarkdownV2); err != nil {
			// If editing fails, log it but don't stop the flow. We can still send the next message.
			log.Printf("Could not edit quiz result message: %v", err)
		}

		if completionResult.UpdatedAchievementID != 0 {
			h.sendAchievementUpdate(c, ctxUser.ID, completionResult.UpdatedAchievementID)
		}

		// 4. Send a NEW message to show the reply keyboard for continuing.
		promptMsg := "آزمون شما تمام شد برای ادامه رو کلمه بعدی بزنید\\."

		// --- NEW LOGIC: SEND COURSE OVERVIEW INSTEAD OF SIMPLE PROMPT ---

		// 4. Fetch the data needed for the course overview screen.
		overview, err := h.getOverview.Handle(context.Background(), courseQueries.GetOverviewQuery{
			UserID:   ctxUser.ID,
			CourseID: fullAttempt.CourseID,
		})
		if err != nil {
			log.Printf("Failed to get course overview after quiz: %v", err)
			// Send a fallback message if we can't get the overview
			_, errSend := c.Bot().Send(c.Chat(), "Quiz complete!", keyboards.BackToCourseListKeyboard())
			return errSend
		}

		// 5. Map the result to the presentation DTO
		overviewDTO := dto.CourseOverview{
			ID:                     overview.ID,
			Title:                  overview.Title,
			PersianTitle:           overview.PersianTitle,
			PersianFullDescription: overview.PersianFullDescription,
			TotalWords:             overview.TotalWords,
			ProgressPercentage:     overview.ProgressPercentage,
			IsCompleted:            overview.IsCompleted,
			IsStarted:              overview.IsStarted,
		}

		kb := keyboards.CourseDetailsKeyboard(overviewDTO)

		// 7. Send the overview as a new message.
		if _, err := c.Bot().Send(c.Chat(), promptMsg, kb, telebot.ModeMarkdownV2); err != nil {
			log.Printf("Could not send course overview prompt: %v", err)
			return err
		}

		// After the quiz is done, set the user's state back to the course details view
		newState := fmt.Sprintf("course_details:%d", fullAttempt.CourseID)
		if err := h.userRepo.UpdateLastMenu(context.Background(), ctxUser.ID, newState); err != nil {
			log.Printf("Failed to update user menu state after quiz: %v", err)
		}

	} else {

		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(res.NextQuestion.Options), func(i, j int) {
			res.NextQuestion.Options[i], res.NextQuestion.Options[j] = res.NextQuestion.Options[j], res.NextQuestion.Options[i]
		})

		questionMsg := formatters.FormatQuizQuestion(res.NextQuestion, fullAttempt.CurrentQuestionIndex, len(fullAttempt.Questions))
		kb := keyboards.QuizQuestionOptionsKeyboard(res.NextQuestion.Options, uint(attemptID))
		_, err = c.Bot().Edit(c.Callback().Message, questionMsg, kb, telebot.ModeMarkdownV2)
	}

	if err != nil {
		log.Printf("Failed to edit message for quiz attempt %d: %v", attemptID, err)
	}
	return nil
}

func (h *Handler) sendAchievementUpdate(c telebot.Context, userID, achievementID uint) {
	ach, err := h.achRepo.FindByID(context.Background(), achievementID)
	if err != nil {
		log.Printf("Could not find achievement %d to send update: %v", achievementID, err)
		return
	}

	userAch, err := h.achRepo.GetUserAchievement(context.Background(), userID, achievementID)
	if err != nil {
		if errors.Is(err, achievement.ErrUserAchNotFound) {
			// This is not an error. It means the user has 0% progress.
			// We must create a new, empty achievement object with a non-nil bitset to represent this.
			userAch = achievement.UserAchievement{
				UserID:        userID,
				AchievementID: achievementID,
				State:         bitset.New(ach.TotalItems), // Creates a new, empty bitset of the correct size
			}
		} else {
			// This is a real database error.
			log.Printf("Could not get user achievement progress for user %d, ach %d: %v", userID, achievementID, err)
			return
		}
	}

	// Now, userAch.State is GUARANTEED to be a valid, non-nil bitset.
	generatedPath, err := h.imgSvc.Generate(ach.ImageURL, userAch.State, ach.GridWidth, ach.GridHeight)
	if err != nil {
		log.Printf("Failed to generate achievement image: %v", err)
		return
	}
	defer os.Remove(generatedPath)

	caption := fmt.Sprintf("🏆 *%s*\n\n`%s`", tgmarkdown.Escape(ach.Title), tgmarkdown.Escape(ach.Description))
	photo := &telebot.Photo{
		File:    telebot.FromDisk(generatedPath),
		Caption: caption,
	}

	if _, err := c.Bot().Send(c.Chat(), photo, telebot.ModeMarkdownV2); err != nil {
		log.Printf("Failed to send achievement photo: %v", err)
	}
}

// handleShowAchievement generates and sends a progressive achievement image.
func (h *Handler) handleShowAchievement(c telebot.Context) error {
	defer c.Respond()

	ctxUser, ok := c.Get("dbUser").(user.User)
	if !ok {
		return c.Send("Error identifying user.")
	}

	data := strings.TrimSpace(c.Callback().Data)
	payload := strings.TrimPrefix(data, ShowAchievementCallbackPrefix)
	achievementID, err := strconv.ParseUint(payload, 10, 32)
	if err != nil {
		log.Printf("Invalid achievement ID in callback payload: %s", payload)
		return nil
	}

	h.sendAchievementUpdate(c, ctxUser.ID, uint(achievementID))
	return nil
}
