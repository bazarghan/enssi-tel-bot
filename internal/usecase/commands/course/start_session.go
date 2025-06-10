package course

import (
	"context"
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
)

// StartSessionCommand defines the input for starting a course session.
type StartSessionCommand struct {
	UserID   uint
	CourseID uint
}

// StartSessionResult tells the presentation layer what to show next.
type StartSessionResult struct {
	NextStep      Step
	Word          word.Word
	MessageToUser string
}

type Step int

const (
	ShowWord Step = iota
	ShowQuiz
	CourseEnded
)

// StartSessionHandler processes the command.
type StartSessionHandler struct {
	courseRepo course.Repository
	wordRepo   word.Repository
}

// NewStartSessionHandler creates a new handler.
func NewStartSessionHandler(courseRepo course.Repository, wordRepo word.Repository) StartSessionHandler {
	return StartSessionHandler{courseRepo: courseRepo, wordRepo: wordRepo}
}

// Handle executes the command.
func (h StartSessionHandler) Handle(ctx context.Context, cmd StartSessionCommand) (StartSessionResult, error) {
	progress, err := h.courseRepo.GetOrCreateUserCourse(ctx, cmd.UserID, cmd.CourseID)
	if err != nil {
		return StartSessionResult{}, fmt.Errorf("could not get or create user course: %w", err)
	}

	if progress.IsCompleted {
		return StartSessionResult{
			NextStep:      CourseEnded,
			MessageToUser: "تبریک! 🎉 شما این دوره را با موفقیت به پایان رساندید.",
		}, nil
	}

	// Placeholder for quiz logic
	// In a future slice, this will check progress and quiz attempt status.
	// For now, we assume no quiz is due at the start of a session.

	// The next word to study is at index `WordsCompleted + 1`.
	nextWordIndex := uint(progress.WordsCompleted + 1)

	nextWord, err := h.wordRepo.FindByCourseIndex(ctx, cmd.CourseID, nextWordIndex)
	if err != nil {
		if errors.Is(err, word.ErrCourseWordLinkNotFound) {
			return StartSessionResult{
				NextStep:      CourseEnded,
				MessageToUser: "شما به پایان کلمات موجود رسیده‌اید. 🏁",
			}, nil
		}
		return StartSessionResult{}, fmt.Errorf("failed to find next word: %w", err)
	}

	return StartSessionResult{
		NextStep: ShowWord,
		Word:     nextWord,
	}, nil
}
