package callback

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	submitCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/quiz"
	"gopkg.in/telebot.v4"
)

const QuizAnswerCallbackPrefix = "quiz_ans:"

// Handler holds dependencies for callback handlers.
type Handler struct {
	submitAnswer submitCmd.SubmitAnswerHandler
	quizRepo     quiz.Repository // For updating message ID
}

// NewHandler creates a new callback handler.
func NewHandler(submitAnswer submitCmd.SubmitAnswerHandler, quizRepo quiz.Repository) *Handler {
	return &Handler{submitAnswer: submitAnswer, quizRepo: quizRepo}
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
