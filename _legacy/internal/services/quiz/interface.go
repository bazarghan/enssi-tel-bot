package quiz

import (
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"github.com/2000ostd/enssi-tel-bot/internal/services/word" // Added for WordStudiedView
)

// QuizService defines the interface for quiz-related operations.
type QuizService interface {
	// StartOrResumeQuiz finds an active quiz for the given user, course, and progress point (block),
	// or creates a new one if none exists or the existing one is completed.
	// userCourseProgress is the progress point that triggers this quiz (for COURSE_BLOCK type).
	StartOrResumeQuiz(userID uint, courseID uint, userCourseProgress uint) (*QuizState, error)

	// CreateReviewQuiz creates a new quiz specifically for reviewing words.
	CreateReviewQuiz(userID uint, wordsToReview []word.WordStudiedView) (*QuizState, error)

	// SubmitAnswer records a user's answer to a quiz question and determines the next step.
	// It returns the state of the next question or the final quiz results if the quiz is completed.
	SubmitAnswer(attemptID uint, chosenOptionID uint, userID uint) (*AnswerSubmissionResult, error)

	// GetQuizResults retrieves the final results of a specific, completed quiz attempt.
	GetQuizResults(attemptID uint, userID uint) (*QuizResult, error)

	// FindAnyActiveQuizAttempt finds any non-completed quiz attempt for a user.
	FindAnyActiveQuizAttempt(userID uint) (*models.QuizAttempt, error)

	// UpdateQuizAttemptMessageID updates the Telegram message ID associated with the current question of a quiz attempt.
	UpdateQuizAttemptMessageID(attemptID uint, messageID int) error

	// GetQuizType retrieves the type of a quiz.
	GetQuizType(quizID uint) (models.QuizType, error)
}
