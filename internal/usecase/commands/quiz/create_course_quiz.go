package quiz

import (
	"context"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
)

const WordsPerQuizBlock = 12

// CreateCourseQuizCommand defines the input.
type CreateCourseQuizCommand struct {
	UserID          uint
	CourseID        uint
	TriggerProgress uint
}

// CreateCourseQuizResult is the output.
type CreateCourseQuizResult struct {
	QuizAttempt quiz.Attempt
}

// CreateCourseQuizHandler creates a quiz for a block of words in a course.
type CreateCourseQuizHandler struct {
	quizRepo quiz.Repository
	wordRepo word.Repository
}

// NewCreateCourseQuizHandler creates a new handler.
func NewCreateCourseQuizHandler(quizRepo quiz.Repository, wordRepo word.Repository) CreateCourseQuizHandler {
	return CreateCourseQuizHandler{quizRepo: quizRepo, wordRepo: wordRepo}
}

// Handle executes the command.
func (h CreateCourseQuizHandler) Handle(ctx context.Context, cmd CreateCourseQuizCommand) (CreateCourseQuizResult, error) {
	// 1. Check if an active attempt already exists.
	activeAttempt, err := h.quizRepo.FindActiveCourseBlockAttempt(ctx, cmd.UserID, cmd.CourseID, cmd.TriggerProgress)
	if err == nil {
		return CreateCourseQuizResult{QuizAttempt: activeAttempt}, nil // Return existing attempt
	}

	// 2. Get word IDs for the block.
	offset := cmd.TriggerProgress - WordsPerQuizBlock
	wordIDs, err := h.wordRepo.FindWordIDsByCourseBlock(ctx, cmd.CourseID, WordsPerQuizBlock, offset)
	if err != nil {
		return CreateCourseQuizResult{}, fmt.Errorf("could not get words for quiz: %w", err)
	}

	// 3. Create quiz structure.
	quizID, err := h.quizRepo.CreateQuiz(ctx, quiz.CourseBlock, cmd.CourseID, cmd.TriggerProgress, wordIDs)
	if err != nil {
		return CreateCourseQuizResult{}, fmt.Errorf("could not create quiz structure: %w", err)
	}

	// 4. Create a new attempt.
	newAttempt, err := h.quizRepo.CreateAttempt(ctx, cmd.UserID, quizID)
	if err != nil {
		return CreateCourseQuizResult{}, fmt.Errorf("could not create quiz attempt: %w", err)
	}

	return CreateCourseQuizResult{QuizAttempt: newAttempt}, nil
}
