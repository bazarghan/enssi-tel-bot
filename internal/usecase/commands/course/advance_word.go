package course

import (
	"context"
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
)

// AdvanceWordCommand defines the input for advancing to the next word.
type AdvanceWordCommand struct {
	UserID   uint
	CourseID uint
}

// AdvanceWordResult tells the presentation layer what to show next.
// It reuses the StartSessionResult structure for consistency.
type AdvanceWordResult = StartSessionResult

// AdvanceWordHandler processes the command.
type AdvanceWordHandler struct {
	courseRepo course.Repository
	wordRepo   word.Repository
}

// NewAdvanceWordHandler creates a new handler.
func NewAdvanceWordHandler(courseRepo course.Repository, wordRepo word.Repository) AdvanceWordHandler {
	return AdvanceWordHandler{courseRepo: courseRepo, wordRepo: wordRepo}
}

// Handle executes the command.
func (h AdvanceWordHandler) Handle(ctx context.Context, cmd AdvanceWordCommand) (AdvanceWordResult, error) {
	// 1. Get current progress (e.g., user has completed N words).
	currentProgress, err := h.courseRepo.GetUserProgress(ctx, cmd.UserID, cmd.CourseID)
	if err != nil {
		return AdvanceWordResult{}, fmt.Errorf("failed to get current progress: %w", err)
	}
	if currentProgress.IsCompleted {
		return AdvanceWordResult{NextStep: CourseEnded, MessageToUser: "دوره قبلا تمام شده است."}, nil
	}

	wordJustStudiedIndex := uint(currentProgress.WordsCompleted)

	// 2. Mark word N as studied using the SRS primitive.
	// This happens after the user has seen it and clicked "Next".
	if wordJustStudiedIndex > 0 {
		wordToMark, err := h.wordRepo.FindByCourseIndex(ctx, cmd.CourseID, wordJustStudiedIndex)
		if err != nil {
			return AdvanceWordResult{}, fmt.Errorf("could not find word at index %d to mark as studied: %w", wordJustStudiedIndex, err)
		}

		studiedWord, err := h.wordRepo.FindStudiedWord(ctx, cmd.UserID, wordToMark.ID)
		if err != nil && !errors.Is(err, word.ErrStudiedWordNotFound) {
			return AdvanceWordResult{}, fmt.Errorf("error checking for studied word: %w", err)
		}

		// If not found, a new StudiedWord struct is created.
		if errors.Is(err, word.ErrStudiedWordNotFound) {
			studiedWord = word.StudiedWord{UserID: cmd.UserID, WordID: wordToMark.ID}
		}

		// In this flow, we assume seeing the word and clicking "Next" means it was correctly recalled.
		studiedWord.CalculateNextReview(true)

		if err := h.wordRepo.SaveStudiedWord(ctx, studiedWord); err != nil {
			return AdvanceWordResult{}, fmt.Errorf("failed to save studied word record: %w", err)
		}
	}

	// 3. Increment course progress (user is now at N+1).
	newProgress, err := h.courseRepo.IncrementProgress(ctx, cmd.UserID, cmd.CourseID)
	if err != nil {
		return AdvanceWordResult{}, fmt.Errorf("failed to increment course progress: %w", err)
	}

	// 4. Determine next step (word N+1, quiz, or end).
	if newProgress.IsCompleted {
		return AdvanceWordResult{
			NextStep:      CourseEnded,
			MessageToUser: "تبریک! 🎉 شما این دوره را با موفقیت به پایان رساندید.",
		}, nil
	}

	// Placeholder for quiz logic
	// if newProgress.WordsCompleted % 12 == 0 { ... return ShowQuiz ... }

	nextWord, err := h.wordRepo.FindByCourseIndex(ctx, cmd.CourseID, uint(newProgress.WordsCompleted+1))
	if err != nil {
		if errors.Is(err, word.ErrCourseWordLinkNotFound) {
			return AdvanceWordResult{
				NextStep:      CourseEnded,
				MessageToUser: "شما به پایان کلمات موجود رسیده‌اید. 🏁",
			}, nil
		}
		return AdvanceWordResult{}, fmt.Errorf("failed to find next word: %w", err)
	}

	return AdvanceWordResult{NextStep: ShowWord, Word: nextWord}, nil
}
