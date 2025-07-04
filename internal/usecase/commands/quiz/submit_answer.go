package quiz

import (
	"context"
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"

	achCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/achievement"
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

	MasteryNotifications []achCmd.MasteryNotification
}

// SubmitAnswerHandler processes a user's quiz answer.
type SubmitAnswerHandler struct {
	logger   logger.Logger
	quizRepo quiz.Repository
	wordRepo word.Repository

	masteryHandler achCmd.HandleWordMasteryHandler
}

// NewSubmitAnswerHandler creates a new handler.
func NewSubmitAnswerHandler(
	appLogger logger.Logger,
	quizRepo quiz.Repository,
	wordRepo word.Repository,
	masteryHandler achCmd.HandleWordMasteryHandler,

) SubmitAnswerHandler {
	return SubmitAnswerHandler{
		logger:         appLogger,
		quizRepo:       quizRepo,
		wordRepo:       wordRepo,
		masteryHandler: masteryHandler,
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

	// We create a variable to hold the notifications.
	var masteryNotifications []achCmd.MasteryNotification
	// We now process the SRS for both correct and incorrect answers in a review quiz.
	if attempt.Type == quiz.Review {
		// We call our helper function and pass `chosenOption.IsCorrect`
		notifications, err := h.updateSRSAndCheckMastery(ctx, cmd.UserID, currentQuestion.WordID, chosenOption.IsCorrect)
		if err != nil {
			// Log the error but don't block the quiz flow
			h.logger.Error("failed to update SRS or check mastery", "error", err)
		}
		masteryNotifications = notifications
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
		return SubmitAnswerResult{
			IsCompleted:          true,
			FinalResult:          finalResult,
			MasteryNotifications: masteryNotifications,
		}, nil
	}

	// Quiz continues, return the next question.
	nextQuestion := attempt.Questions[attempt.CurrentQuestionIndex+1]
	return SubmitAnswerResult{
		IsCompleted:          false,
		NextQuestion:         nextQuestion,
		MasteryNotifications: masteryNotifications,
	}, nil
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

// REPLACE the previous helper method with this corrected version
func (h *SubmitAnswerHandler) updateSRSAndCheckMastery(ctx context.Context, userID, wordID uint, wasCorrect bool) ([]achCmd.MasteryNotification, error) {
	if wordID == 0 {
		return nil, nil
	}

	studiedWord, err := h.wordRepo.FindStudiedWord(ctx, userID, wordID)
	if err != nil {
		return nil, fmt.Errorf("could not find studied word to update SRS: %w", err)
	}

	// --- START of Corrected Logic ---
	previousInterval := studiedWord.ReviewIntervalDays
	// This now correctly handles BOTH true and false answers, resetting the interval on wrong answers.
	studiedWord.CalculateNextReview(wasCorrect)

	if err := h.wordRepo.SaveStudiedWord(ctx, studiedWord); err != nil {
		return nil, fmt.Errorf("failed to save studied word: %w", err)
	}

	// We ONLY check for mastery achievements if the answer was correct.
	if wasCorrect {
		// If the interval just crossed the mastery threshold, call the mastery handler.
		if studiedWord.ReviewIntervalDays >= 8 && previousInterval < 8 {
			masteryCmd := achCmd.HandleWordMasteryCommand{UserID: userID}
			return h.masteryHandler.Handle(ctx, masteryCmd)
		}
	}
	// --- END of Corrected Logic ---

	return nil, nil
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
