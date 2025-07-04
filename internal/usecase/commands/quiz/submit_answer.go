package quiz

import (
	"context"
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
)

// SubmitAnswerCommand defines the input.
type SubmitAnswerCommand struct {
	AttemptID uint
	OptionID  uint
	UserID    uint
}

// SubmitAnswerResult tells the presentation layer what to do next.
type SubmitAnswerResult struct {
	IsCompleted  bool
	NextQuestion quiz.Question
	FinalResult  quiz.Result
}

// SubmitAnswerHandler processes a user's quiz answer.
type SubmitAnswerHandler struct {
	logger   logger.Logger
	quizRepo quiz.Repository
	wordRepo word.Repository
}

// NewSubmitAnswerHandler creates a new handler.
func NewSubmitAnswerHandler(
	appLogger logger.Logger,
	quizRepo quiz.Repository,
	wordRepo word.Repository,
) SubmitAnswerHandler {
	return SubmitAnswerHandler{
		logger:   appLogger,
		quizRepo: quizRepo,
		wordRepo: wordRepo,
	}
}

// Handle executes the command.
func (h SubmitAnswerHandler) Handle(ctx context.Context, cmd SubmitAnswerCommand) (SubmitAnswerResult, error) {
	attempt, err := h.quizRepo.GetAttempt(ctx, cmd.AttemptID)
	if err != nil {
		return SubmitAnswerResult{}, err
	}

	if err := h.validateAttempt(attempt, cmd.UserID); err != nil {
		return SubmitAnswerResult{}, err
	}

	currentQuestion := attempt.Questions[attempt.CurrentQuestionIndex]
	chosenOption, err := h.validateOption(currentQuestion, cmd.OptionID)
	if err != nil {
		return SubmitAnswerResult{}, err
	}

	if err := h.quizRepo.SaveAnswer(ctx, attempt.ID, currentQuestion.ID, chosenOption.ID, chosenOption.IsCorrect); err != nil {
		if errors.Is(err, quiz.ErrQuestionAlreadyAnswered) {
			return SubmitAnswerResult{}, err // Propagate specific error for handler to ignore
		}
		return SubmitAnswerResult{}, fmt.Errorf("failed to save answer: %w", err)
	}

	if attempt.Type == quiz.Review {
		h.updateSRS(ctx, cmd.UserID, currentQuestion.WordID, chosenOption.IsCorrect)
	}

	if err := h.quizRepo.IncrementQuestionIndex(ctx, attempt.ID); err != nil {
		return SubmitAnswerResult{}, fmt.Errorf("failed to advance attempt: %w", err)
	}

	// Check if the quiz is now complete.
	if attempt.CurrentQuestionIndex+1 >= len(attempt.Questions) {
		finalScore, err := h.quizRepo.MarkAttemptCompleted(ctx, attempt.ID)
		if err != nil {
			return SubmitAnswerResult{}, fmt.Errorf("failed to mark attempt complete: %w", err)
		}

		finalResult := h.calculateFinalResult(attempt, finalScore)
		return SubmitAnswerResult{IsCompleted: true, FinalResult: finalResult}, nil
	}

	// Quiz continues, return the next question.
	nextQuestion := attempt.Questions[attempt.CurrentQuestionIndex+1]
	return SubmitAnswerResult{IsCompleted: false, NextQuestion: nextQuestion}, nil
}

func (h *SubmitAnswerHandler) validateAttempt(attempt quiz.Attempt, userID uint) error {
	if attempt.UserID != userID {
		return quiz.ErrUserMismatch
	}
	if attempt.IsCompleted {
		return quiz.ErrAttemptAlreadyCompleted
	}
	if attempt.CurrentQuestionIndex >= len(attempt.Questions) {
		return quiz.ErrAttemptAlreadyCompleted
	}
	return nil
}

func (h *SubmitAnswerHandler) validateOption(question quiz.Question, optionID uint) (quiz.Option, error) {
	for _, opt := range question.Options {
		if opt.ID == optionID {
			return opt, nil
		}
	}
	return quiz.Option{}, quiz.ErrInvalidOption
}

func (h *SubmitAnswerHandler) updateSRS(ctx context.Context, userID, wordID uint, wasCorrect bool) {
	if wordID == 0 {
		return
	}
	studiedWord, err := h.wordRepo.FindStudiedWord(ctx, userID, wordID)
	if err != nil {
		h.logger.Warn("Could not find studied word to update SRS", "wordID", wordID, "userID", userID, "error", err)
		return
	}
	studiedWord.CalculateNextReview(wasCorrect)
	h.wordRepo.SaveStudiedWord(ctx, studiedWord)
}

func (h *SubmitAnswerHandler) calculateFinalResult(attempt quiz.Attempt, finalScore int) quiz.Result {
	result := quiz.Result{
		Score:           finalScore,
		TotalQuestions:  len(attempt.Questions),
		CourseID:        attempt.CourseID,
		TriggerProgress: attempt.TriggerProgress,
	}

	if attempt.Type == quiz.CourseBlock {
		result.Passed = result.Score >= quiz.QuizPassThreshold
		if !result.Passed {
			result.ShouldResetProgress = true
			if attempt.CourseID > 0 {
				// This logic can be refined, but for now, it resets to the block start.
				if result.TriggerProgress >= WordsPerQuizBlock {
					result.SuggestedNewProgress = result.TriggerProgress - WordsPerQuizBlock
				} else {
					result.SuggestedNewProgress = 0
				}
			}
		}
	} else {
		result.Passed = true // Review quizzes always "pass" in terms of flow.
	}

	return result
}
