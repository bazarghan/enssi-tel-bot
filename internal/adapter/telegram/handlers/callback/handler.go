package callback

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/keyboards"

	"github.com/2000ostd/enssi-tel-bot/internal/domain/achievement"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	submitCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/quiz"
	"gopkg.in/telebot.v4"
)

const (
	QuizAnswerCallbackPrefix      = "quiz_ans:"
	ShowAchievementCallbackPrefix = "ach_show:"
)

// Handler holds dependencies for callback handlers.
type Handler struct {
	submitAnswer submitCmd.SubmitAnswerHandler
	quizRepo     quiz.Repository
	achRepo      achievement.Repository
	imgSvc       achievement.ImageGenerator
}

// NewHandler creates a new callback handler.
func NewHandler(
	submitAnswer submitCmd.SubmitAnswerHandler,
	quizRepo quiz.Repository,
	achRepo achievement.Repository,
	imgSvc achievement.ImageGenerator,
) *Handler {
	return &Handler{
		submitAnswer: submitAnswer,
		quizRepo:     quizRepo,
		achRepo:      achRepo,
		imgSvc:       imgSvc,
	}
}

// Handle routes incoming callback queries.
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

func (h *Handler) handleQuizAnswer(c telebot.Context) error {
	defer c.Respond()

	ctxUser, ok := c.Get("dbUser").(user.User)
	if !ok {
		return c.Send("خطای شناسایی کاربر.")
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
		_, err := c.Bot().Edit(c.Callback().Message, "خطایی در پردازش پاسخ شما رخ داد.")
		return err
	}

	if res.IsCompleted {
		finalMsg := formatters.FormatQuizResult(res.FinalResult)
		_, err = c.Bot().Edit(c.Callback().Message, finalMsg, telebot.ModeMarkdownV2)
	} else {
		questionMsg := formatters.FormatQuizQuestion(res.NextQuestion, int(attemptID))
		kb := keyboards.QuizQuestionOptionsKeyboard(res.NextQuestion.Options, uint(attemptID))
		_, err = c.Bot().Edit(c.Callback().Message, questionMsg, kb, telebot.ModeMarkdownV2)
	}

	if err != nil {
		log.Printf("Failed to edit message for quiz attempt %d: %v", attemptID, err)
	}
	return nil
}

func (h *Handler) handleShowAchievement(c telebot.Context) error {
	defer c.Respond()

	ctxUser, ok := c.Get("dbUser").(user.User)
	if !ok {
		return c.Send("خطای شناسایی کاربر.")
	}

	payload := strings.TrimPrefix(c.Callback().Data, ShowAchievementCallbackPrefix)
	achievementID, err := strconv.ParseUint(payload, 10, 32)
	if err != nil {
		log.Printf("Invalid achievement ID in callback payload: %s", payload)
		return nil
	}

	// 1. Get base achievement details (like image path).
	ach, err := h.achRepo.FindByID(context.Background(), uint(achievementID))
	if err != nil {
		log.Printf("Could not find achievement %d: %v", achievementID, err)
		return c.Send("اطلاعات این دستاورد یافت نشد.")
	}

	// 2. Get user's specific progress for this achievement.
	userAch, err := h.achRepo.GetUserAchievement(context.Background(), ctxUser.ID, uint(achievementID))
	if err != nil && !errors.Is(err, achievement.ErrUserAchNotFound) {
		log.Printf("Could not get user achievement progress for user %d, ach %d: %v", ctxUser.ID, achievementID, err)
		return c.Send("خطا در دریافت اطلاعات پیشرفت شما.")
	}

	// 3. Generate the progressive image.
	generatedPath, err := h.imgSvc.Generate(ach.ImageURL, userAch.State, ach.GridWidth, ach.GridHeight)
	if err != nil {
		log.Printf("Failed to generate achievement image: %v", err)
		return c.Send("خطا در ساخت تصویر دستاورد.")
	}
	defer os.Remove(generatedPath) // Clean up the temp file.

	// 4. Send the image.
	photo := &telebot.Photo{
		File:    telebot.FromDisk(generatedPath),
		Caption: fmt.Sprintf("🏆 *%s*\n\n_%s_", ach.Title, ach.Description),
	}

	_, err = c.Bot().Send(c.Chat(), photo, telebot.ModeMarkdown)
	if err != nil {
		log.Printf("Failed to send achievement photo: %v", err)
	}

	return nil
}
