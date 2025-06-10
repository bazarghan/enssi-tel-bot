package quiz

import (
	"context"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
)

const QuizPassThreshold = 9

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
	quizRepo quiz.Repository
	wordRepo word.Repository
}

// NewSubmitAnswerHandler creates a new handler.
func NewSubmitAnswerHandler(quizRepo quiz.Repository, wordRepo word.Repository) SubmitAnswerHandler {
	return SubmitAnswerHandler{quizRepo: quizRepo, wordRepo: wordRepo}
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
		// This could be a duplicate answer, which we can ignore or handle. For now, we return the error.
		return SubmitAnswerResult{}, fmt.Errorf("failed to save answer: %w", err)
	}

	if attempt.Type == quiz.Review {
		h.updateSRS(ctx, cmd.UserID, currentQuestion.WordID, chosenOption.IsCorrect)
	}

	if err := h.quizRepo.IncrementQuestionIndex(ctx, attempt.ID); err != nil {
		return SubmitAnswerResult{}, fmt.Errorf("failed to advance attempt: %w", err)
	}

	attempt.CurrentQuestionIndex++

	// Check if the quiz is now complete.
	if attempt.CurrentQuestionIndex >= len(attempt.Questions) {
		h.quizRepo.MarkAttemptCompleted(ctx, attempt.ID)

		finalResult := h.calculateFinalResult(ctx, attempt)
		return SubmitAnswerResult{IsCompleted: true, FinalResult: finalResult}, nil
	}

	// Quiz continues, return the next question.
	nextQuestion := attempt.Questions[attempt.CurrentQuestionIndex]
	return SubmitAnswerResult{IsCompleted: false, NextQuestion: nextQuestion}, nil
}

func (h *SubmitAnswerHandler) validateAttempt(attempt quiz.Attempt, userID uint) error {
	if attempt.UserID != userID {
		return quiz.ErrUserMismatch
	}
	if attempt.IsCompleted {
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
	studiedWord, err := h.wordRepo.FindStudiedWord(ctx, userID, wordID)
	if err != nil {
		// Log error but don't fail the entire operation.
		fmt.Printf("could not find studied word %d for user %d to update SRS: %v\n", wordID, userID, err)
		return
	}
	studiedWord.CalculateNextReview(wasCorrect)
	h.wordRepo.SaveStudiedWord(ctx, studiedWord)
}

func (h *SubmitAnswerHandler) calculateFinalResult(ctx context.Context, attempt quiz.Attempt) quiz.Result {
	// Re-fetch attempt to get latest score after all answers are in.
	finalAttempt, _ := h.quizRepo.GetAttempt(ctx, attempt.ID)

	result := quiz.Result{
		Score:          finalAttempt.Score,
		TotalQuestions: len(finalAttempt.Questions),
	}

	if finalAttempt.Type == quiz.CourseBlock {
		result.Passed = result.Score >= QuizPassThreshold
		if !result.Passed {
			result.ShouldResetProgress = true
			if finalAttempt.CourseID > 0 && WordsPerQuizBlock > 0 { // Placeholder for TriggerProgress
				result.SuggestedNewProgress = 0 //Simplified logic
			}
		}
	} else {
		result.Passed = true // Review quizzes always "pass" in terms of flow.
	}

	return result
}
