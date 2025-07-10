package quiz

import (
	"context"
	"fmt"
	"github.com/bazarghan/enssi-tel-bot/internal/domain/quiz"
	"github.com/bazarghan/enssi-tel-bot/internal/domain/word"
)

// CreateReviewQuizCommand defines the input for creating a review quiz.
type CreateReviewQuizCommand struct {
	UserID        uint
	WordsToReview []word.StudiedWord
}

// CreateReviewQuizResult holds the newly created quiz attempt.
type CreateReviewQuizResult struct {
	QuizAttempt quiz.Attempt
}

// CreateReviewQuizHandler creates a quiz for reviewing words from the SRS.
type CreateReviewQuizHandler struct {
	quizRepo quiz.Repository
}

// NewCreateReviewQuizHandler creates a new handler.
func NewCreateReviewQuizHandler(quizRepo quiz.Repository) CreateReviewQuizHandler {
	return CreateReviewQuizHandler{quizRepo: quizRepo}
}

// Handle executes the command.
func (h CreateReviewQuizHandler) Handle(ctx context.Context, cmd CreateReviewQuizCommand) (CreateReviewQuizResult, error) {
	if len(cmd.WordsToReview) == 0 {
		return CreateReviewQuizResult{}, quiz.ErrNoWordsForQuiz
	}

	wordIDs := make([]uint, len(cmd.WordsToReview))
	for i, w := range cmd.WordsToReview {
		wordIDs[i] = w.WordID
	}

	// 1. Create the quiz structure (questions and options).
	quizID, err := h.quizRepo.CreateQuiz(ctx, quiz.Review, 0, 0, wordIDs)
	if err != nil {
		return CreateReviewQuizResult{}, fmt.Errorf("could not create review quiz structure: %w", err)
	}

	// 2. Create the attempt for the user.
	newAttempt, err := h.quizRepo.CreateAttempt(ctx, cmd.UserID, quizID)
	if err != nil {
		return CreateReviewQuizResult{}, fmt.Errorf("could not create review quiz attempt: %w", err)
	}

	return CreateReviewQuizResult{QuizAttempt: newAttempt}, nil
}
