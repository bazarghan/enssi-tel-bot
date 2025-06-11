package callback

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/achievement"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"

	courseCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/course"
	submitCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/quiz"

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
}

// NewHandler creates a new callback handler with all its dependencies.
func NewHandler(
	submitAnswer submitCmd.SubmitAnswerHandler,
	handleQuizCompletion courseCmd.HandleQuizCompletionHandler,
	quizRepo quiz.Repository,
	achRepo achievement.Repository,
	imgSvc achievement.ImageGenerator,
) *Handler {
	return &Handler{
		submitAnswer:         submitAnswer,
		handleQuizCompletion: handleQuizCompletion,
		quizRepo:             quizRepo,
		achRepo:              achRepo,
		imgSvc:               imgSvc,
	}
}

// Handle routes incoming callback queries to the appropriate sub-handler.
func (h *Handler) Handle(c telebot.Context) error {
	cb := c.Callback()
	if cb == nil {
		return nil
	}

	if strings.HasPrefix(cb.Data, QuizAnswerCallbackPrefix) {
		return h.handleQuizAnswer(c)
	}

	if strings.HasPrefix(cb.Data, ShowAchievementCallbackPrefix) {
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

	payload := strings.TrimPrefix(c.Callback().Data, QuizAnswerCallbackPrefix)
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
		completionCmd := courseCmd.HandleQuizCompletionCommand{
			UserID:   ctxUser.ID,
			CourseID: fullAttempt.CourseID,
			Result:   res.FinalResult,
		}
		courseRes, err := h.handleQuizCompletion.Handle(context.Background(), completionCmd)
		if err != nil {
			log.Printf("Error handling quiz completion for attempt %d: %v", attemptID, err)
			_, err = c.Bot().Edit(c.Callback().Message, "Error processing quiz result.")
			return err
		}

		// Now, show the next step from the course flow
		if courseRes.NextStep == courseCmd.ShowWord {
			msg := formatters.FormatWordForDisplay(courseRes.Word)
			kb := keyboards.InCourseNavigationKeyboard()
			_, err = c.Bot().Edit(c.Callback().Message, msg, kb, telebot.ModeMarkdownV2)
		} else { // CourseEnded or another state
			msg := tgmarkdown.Escape(courseRes.MessageToUser)
			kb := keyboards.BackToCourseListKeyboard()
			_, err = c.Bot().Edit(c.Callback().Message, msg, kb, telebot.ModeMarkdownV2)
		}
	} else {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(res.NextQuestion.Options), func(i, j int) {
			res.NextQuestion.Options[i], res.NextQuestion.Options[j] = res.NextQuestion.Options[j], res.NextQuestion.Options[i]
		})

		questionMsg := formatters.FormatQuizQuestion(res.NextQuestion, fullAttempt.CurrentQuestionIndex+1, len(fullAttempt.Questions))
		kb := keyboards.QuizQuestionOptionsKeyboard(res.NextQuestion.Options, uint(attemptID))
		_, err = c.Bot().Edit(c.Callback().Message, questionMsg, kb, telebot.ModeMarkdownV2)
	}

	if err != nil {
		log.Printf("Failed to edit message for quiz attempt %d: %v", attemptID, err)
	}
	return nil
}

// handleShowAchievement generates and sends a progressive achievement image.
func (h *Handler) handleShowAchievement(c telebot.Context) error {
	defer c.Respond()

	ctxUser, ok := c.Get("dbUser").(user.User)
	if !ok {
		return c.Send("Error identifying user.")
	}

	payload := strings.TrimPrefix(c.Callback().Data, ShowAchievementCallbackPrefix)
	achievementID, err := strconv.ParseUint(payload, 10, 32)
	if err != nil {
		log.Printf("Invalid achievement ID in callback payload: %s", payload)
		return nil
	}

	ach, err := h.achRepo.FindByID(context.Background(), uint(achievementID))
	if err != nil {
		log.Printf("Could not find achievement %d: %v", achievementID, err)
		return c.Send("Achievement details not found.")
	}

	userAch, err := h.achRepo.GetUserAchievement(context.Background(), ctxUser.ID, uint(achievementID))
	if err != nil && !errors.Is(err, achievement.ErrUserAchNotFound) {
		log.Printf("Could not get user achievement progress for user %d, ach %d: %v", ctxUser.ID, achievementID, err)
		return c.Send("Error retrieving your progress.")
	}

	generatedPath, err := h.imgSvc.Generate(ach.ImageURL, userAch.State, ach.GridWidth, ach.GridHeight)
	if err != nil {
		log.Printf("Failed to generate achievement image: %v", err)
		return c.Send("Error creating achievement image.")
	}
	defer os.Remove(generatedPath)

	caption := fmt.Sprintf("🏆 *%s*\n\n_%s_", tgmarkdown.Escape(ach.Title), tgmarkdown.Escape(ach.Description))
	photo := &telebot.Photo{
		File:    telebot.FromDisk(generatedPath),
		Caption: caption,
	}

	_, err = c.Bot().Send(c.Chat(), photo, telebot.ModeMarkdownV2)
	if err != nil {
		log.Printf("Failed to send achievement photo: %v", err)
	}

	return nil
}
