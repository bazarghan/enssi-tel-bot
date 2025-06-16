package quiz

import (
	"context"
)

// Repository defines the persistence port for quiz-related data.
type Repository interface {
	// CreateQuiz creates the quiz structure (questions and options).
	CreateQuiz(ctx context.Context, quizType QuizType, courseID, triggerProgress uint, wordIDs []uint) (uint, error)

	// FindActiveCourseBlockAttempt retrieves an incomplete attempt for a specific course block.
	FindActiveCourseBlockAttempt(ctx context.Context, userID, courseID uint, triggerProgress uint) (Attempt, error)

	// GetAttempt fetches a quiz attempt by its ID, including its questions and options.
	GetAttempt(ctx context.Context, attemptID uint) (Attempt, error)

	// CreateAttempt creates a new quiz attempt record for a user.
	CreateAttempt(ctx context.Context, userID, quizID uint) (Attempt, error)

	// SaveAnswer records a user's answer and updates the attempt's score.
	SaveAnswer(ctx context.Context, attemptID, questionID, optionID uint, isCorrect bool) error

	// IncrementQuestionIndex moves the attempt to the next question.
	IncrementQuestionIndex(ctx context.Context, attemptID uint) error

	// MarkAttemptCompleted finalizes an attempt and calculates the final score.
	MarkAttemptCompleted(ctx context.Context, attemptID uint) (score int, err error)

	// UpdateMessageID updates the telegram message ID for the current question.
	UpdateMessageID(ctx context.Context, attemptID uint, messageID int) error

	// Returns the highest score for a completed attempt on a course block. Returns 0 if no completed attempt is found.
	GetHighestScoreForCourseBlock(ctx context.Context, userID, courseID, triggerProgress uint) (int, error)

	// Returns the pendingReview Attempt
	FindPendingReviewAttempt(ctx context.Context, userID uint) (Attempt, error)

	// Delete Attempt by attemptID
	DeleteAttempt(ctx context.Context, attemptID uint) error
}
