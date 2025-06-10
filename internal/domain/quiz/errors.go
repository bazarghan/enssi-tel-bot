package quiz

import "errors"

// Domain-specific errors for quiz operations.
var (
	ErrAttemptNotFound         = errors.New("quiz attempt not found")
	ErrAttemptAlreadyCompleted = errors.New("quiz attempt already completed")
	ErrQuestionAlreadyAnswered = errors.New("question has already been answered")
	ErrInvalidOption           = errors.New("invalid option selected")
	ErrUserMismatch            = errors.New("user does not own this quiz attempt")
	ErrNoWordsForQuiz          = errors.New("not enough words available to create a quiz")
	ErrQuizCreation            = errors.New("failed to create quiz structure")
)
