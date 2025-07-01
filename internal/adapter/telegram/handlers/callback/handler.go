package callback

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	"github.com/bits-and-blooms/bitset"

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
	logger               logger.Logger
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
	appLogger logger.Logger,
	submitAnswer submitCmd.SubmitAnswerHandler,
	handleQuizCompletion courseCmd.HandleQuizCompletionHandler,
	quizRepo quiz.Repository,
	achRepo achievement.Repository,
	imgSvc achievement.ImageGenerator,
	userRepo user.Repository,
	getOverview courseQueries.GetOverviewHandler,
) *Handler {
	return &Handler{
		logger:               appLogger.With("handler", "callback"),
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
		// This can happen in rare cases, not an error.
		return nil
	}

	data := strings.TrimSpace(cb.Data)
	h.logger.Info("Routing callback query", "data", data, "sender_id", c.Sender().ID)

	if strings.HasPrefix(data, QuizAnswerCallbackPrefix) {
		return h.handleQuizAnswer(c)
	}

	if strings.HasPrefix(data, ShowAchievementCallbackPrefix) {
		return h.handleShowAchievement(c)
	}

	h.logger.Warn("Unhandled callback data", "data", cb.Data, "sender_id", c.Sender().ID)
	return c.Respond()
}

// handleQuizAnswer processes a user's answer to a quiz question.
func (h *Handler) handleQuizAnswer(c telebot.Context) error {
	defer c.Respond()

	ctxUser, ok := c.Get("dbUser").(user.User)
	if !ok {
		h.logger.Error("Failed to identify user from context")
		return c.Send("Error identifying user.")
	}

	logger := h.logger.With("user_id", ctxUser.ID, "telegram_id", c.Sender().ID)

	// The telegram callbacks often have whitespace
	data := strings.TrimSpace(c.Callback().Data)

	payload := strings.TrimPrefix(data, QuizAnswerCallbackPrefix)
	parts := strings.Split(payload, ":")
	if len(parts) != 2 {
		logger.Warn("Invalid quiz answer payload", "payload", payload)
		return nil
	}

	attemptID, _ := strconv.ParseUint(parts[0], 10, 64)
	optionID, _ := strconv.ParseUint(parts[1], 10, 64)

	logger = logger.With("attempt_id", attemptID, "option_id", optionID)
	logger.Info("Handling quiz answer")

	cmd := submitCmd.SubmitAnswerCommand{
		AttemptID: uint(attemptID),
		OptionID:  uint(optionID),
		UserID:    ctxUser.ID,
	}

	res, err := h.submitAnswer.Handle(context.Background(), cmd)
	if err != nil {
		if errors.Is(err, quiz.ErrQuestionAlreadyAnswered) {
			logger.Info("Question already answered, ignoring duplicate submission", "error", err)
			return nil // Silently ignore duplicate clicks
		}
		logger.Error("Error submitting quiz answer", "error", err)
		_, errEdit := c.Bot().Edit(c.Callback().Message, "An error occurred while processing your answer.")
		return errEdit
	}

	fullAttempt, err := h.quizRepo.GetAttempt(context.Background(), uint(attemptID))
	if err != nil {
		logger.Error("Could not fetch full attempt for formatting", "error", err)
	}
	if res.IsCompleted {
		logger.Info("Quiz completed")
		// --- NEW: Check if the completed quiz was a Daily Review ---
		if fullAttempt.Type == quiz.Review {
			// 1. Mark the user's daily review as completed for today.
			if err := h.userRepo.UpdateLastReviewSession(context.Background(), ctxUser.ID); err != nil {
				logger.Error("Failed to update last review session", "error", err)
			}

			// 2. Format and show the results, just like a normal quiz.
			resultMsg := formatters.FormatQuizResult(res.FinalResult, fullAttempt)
			if _, err := c.Bot().Edit(c.Callback().Message, resultMsg, telebot.ModeMarkdownV2); err != nil {
				logger.Error("Could not edit review quiz result message", "error", err)
			}

			// 3.Build the main menu, explicitly passing `false` for hasPendingReview.
			mainMenuKeyboard := keyboards.NewMainMenu(ctxUser.IsAdmin, false)

			// 4. Send the final confirmation message with the new, clean main menu.
			if _, err := c.Bot().Send(c.Chat(), "آزمون مرور شما به پایان رسید!", mainMenuKeyboard); err != nil {
				logger.Error("Failed to send final review completion message", "error", err)
			}

			// 5.Update the user's state back to 'main' so they are no longer "in_quiz".
			if err := h.userRepo.UpdateLastMenu(context.Background(), ctxUser.ID, "main"); err != nil {
				logger.Error("Failed to update user menu state after review quiz", "error", err)
			}
			logger.Info("Daily review quiz finished successfully")
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
			logger.Error("Error handling quiz completion", "error", err, "course_id", fullAttempt.CourseID)
			_, err = c.Bot().Edit(c.Callback().Message, "Error processing quiz result.")
			return err
		}

		// 1. Format the result message using your existing formatter.
		resultMsg := formatters.FormatQuizResult(res.FinalResult, fullAttempt)

		// 3. Edit the original quiz message to show the results.
		if _, err := c.Bot().Edit(c.Callback().Message, resultMsg, telebot.ModeMarkdownV2); err != nil {
			logger.Warn("Could not edit quiz result message, proceeding to next step", "error", err)
		}

		if completionResult.UpdatedAchievementID != 0 {
			h.sendAchievementUpdate(c, ctxUser.ID, completionResult.UpdatedAchievementID)
		}

		promptMsg := "آزمون شما تمام شد برای ادامه رو کلمه بعدی بزنید\\."

		// 4. Fetch the data needed for the course overview screen.
		overview, err := h.getOverview.Handle(context.Background(), courseQueries.GetOverviewQuery{
			UserID:   ctxUser.ID,
			CourseID: fullAttempt.CourseID,
		})
		if err != nil {
			logger.Error("Failed to get course overview after quiz", "error", err, "course_id", fullAttempt.CourseID)
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
			logger.Error("Could not send course overview prompt", "error", err)
			return err
		}

		// After the quiz is done, set the user's state back to the course details view
		newState := fmt.Sprintf("course_details:%d", fullAttempt.CourseID)
		if err := h.userRepo.UpdateLastMenu(context.Background(), ctxUser.ID, newState); err != nil {
			logger.Error("Failed to update user menu state after quiz", "new_state", newState, "error", err)
		}
		logger.Info("Standard quiz finished successfully")

	} else {
		logger.Info("Advancing to next question")
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(res.NextQuestion.Options), func(i, j int) {
			res.NextQuestion.Options[i], res.NextQuestion.Options[j] = res.NextQuestion.Options[j], res.NextQuestion.Options[i]
		})

		questionMsg := formatters.FormatQuizQuestion(res.NextQuestion, fullAttempt.CurrentQuestionIndex, len(fullAttempt.Questions))
		kb := keyboards.QuizQuestionOptionsKeyboard(res.NextQuestion.Options, uint(attemptID))
		_, err = c.Bot().Edit(c.Callback().Message, questionMsg, kb, telebot.ModeMarkdownV2)
	}

	if err != nil {
		logger.Error("Failed to edit message for next quiz question", "error", err)
	}
	return nil
}

func (h *Handler) sendAchievementUpdate(c telebot.Context, userID, achievementID uint) {
	logger := h.logger.With("user_id", userID, "achievement_id", achievementID)
	logger.Info("Sending achievement update")

	ach, err := h.achRepo.FindByID(context.Background(), achievementID)
	if err != nil {
		logger.Error("Could not find achievement to send update", "error", err)
		return
	}

	userAch, err := h.achRepo.GetUserAchievement(context.Background(), userID, achievementID)
	if err != nil {
		if errors.Is(err, achievement.ErrUserAchNotFound) {
			logger.Info("User has no prior progress for this achievement, creating new state")
			userAch = achievement.UserAchievement{
				UserID:        userID,
				AchievementID: achievementID,
				State:         bitset.New(ach.TotalItems),
			}
		} else {
			logger.Error("Could not get user achievement progress", "error", err)
			return
		}
	}

	generatedPath, err := h.imgSvc.Generate(ach.ImageURL, userAch.State, ach.GridWidth, ach.GridHeight)
	if err != nil {
		logger.Error("Failed to generate achievement image", "error", err, "image_url", ach.ImageURL)
		return
	}
	defer os.Remove(generatedPath)

	caption := fmt.Sprintf("🏆 *%s*\n\n`%s`", tgmarkdown.Escape(ach.Title), tgmarkdown.Escape(ach.Description))
	photo := &telebot.Photo{
		File:    telebot.FromDisk(generatedPath),
		Caption: caption,
	}

	if _, err := c.Bot().Send(c.Chat(), photo, telebot.ModeMarkdownV2); err != nil {
		logger.Error("Failed to send achievement photo", "error", err)
	} else {
		logger.Info("Successfully sent achievement photo")
	}
}

// handleShowAchievement generates and sends a progressive achievement image.
func (h *Handler) handleShowAchievement(c telebot.Context) error {
	defer c.Respond()

	ctxUser, ok := c.Get("dbUser").(user.User)
	if !ok {
		h.logger.Error("Failed to identify user from context")
		return c.Send("Error identifying user.")
	}
	logger := h.logger.With("user_id", ctxUser.ID, "telegram_id", c.Sender().ID)

	data := strings.TrimSpace(c.Callback().Data)
	payload := strings.TrimPrefix(data, ShowAchievementCallbackPrefix)
	achievementID, err := strconv.ParseUint(payload, 10, 32)
	if err != nil {
		logger.Warn("Invalid achievement ID in callback payload", "payload", payload, "error", err)
		return nil
	}
	logger.Info("Handling show achievement request", "achievement_id", achievementID)

	h.sendAchievementUpdate(c, ctxUser.ID, uint(achievementID))
	return nil
}

