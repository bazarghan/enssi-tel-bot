package course

import (
	"context"
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
	"log"
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
	// First, mark the word the user just saw as studied.
	progress, err := h.courseRepo.GetUserProgress(ctx, cmd.UserID, cmd.CourseID)
	if err != nil {
		return AdvanceWordResult{}, fmt.Errorf("failed to get current progress before advancing: %w", err)
	}
	wordJustStudiedIndex := uint(progress.WordsCompleted)

	// Increment progress to the next word.
	newProgress, err := h.courseRepo.IncrementProgress(ctx, cmd.UserID, cmd.CourseID)
	if err != nil {
		return AdvanceWordResult{}, fmt.Errorf("failed to increment course progress: %w", err)
	}

	// Now update the SRS for the word they just finished.
	if wordJustStudiedIndex > 0 {
		wordToMark, err := h.wordRepo.FindByCourseIndex(ctx, cmd.CourseID, wordJustStudiedIndex)
		if err != nil {
			log.Printf("could not find word at index %d to mark as studied: %v", wordJustStudiedIndex, err)
		} else {
			studiedWord, err := h.wordRepo.FindStudiedWord(ctx, cmd.UserID, wordToMark.ID)
			if err != nil && !errors.Is(err, word.ErrStudiedWordNotFound) {
				log.Printf("error checking for studied word %d: %v", wordToMark.ID, err)
			} else {
				if errors.Is(err, word.ErrStudiedWordNotFound) {
					studiedWord = word.StudiedWord{UserID: cmd.UserID, WordID: wordToMark.ID}
				}
				studiedWord.CalculateNextReview(true) // Assume correct recall as user is advancing.
				if err := h.wordRepo.SaveStudiedWord(ctx, studiedWord); err != nil {
					log.Printf("failed to save studied word record for word %d: %v", wordToMark.ID, err)
				}
			}
		}
	}

	// 4. Determine next step (word N+1, quiz, or end).
	if newProgress.IsCompleted {
		return AdvanceWordResult{
			NextStep:      CourseEnded,
			MessageToUser: "تبریک! 🎉 شما این دوره را با موفقیت به پایان رساندید.",
		}, nil
	}

	// TODO: Check for quiz trigger at newProgress.WordsCompleted
	// if newProgress.WordsCompleted % 12 == 0 { return ShowQuiz }

	nextWordToShowIndex := uint(newProgress.WordsCompleted + 1)
	nextWord, err := h.wordRepo.FindByCourseIndex(ctx, cmd.CourseID, nextWordToShowIndex)
	if err != nil {
		if errors.Is(err, word.ErrCourseWordLinkNotFound) {
			// This might mean we're at the very end and the next thing is the final quiz.
			// For now, we treat it as the end of content.
			return AdvanceWordResult{
				NextStep:      CourseEnded,
				MessageToUser: "شما به پایان کلمات موجود رسیده‌اید. 🏁",
			}, nil
		}
		return AdvanceWordResult{}, fmt.Errorf("failed to find next word: %w", err)
	}

	return AdvanceWordResult{NextStep: ShowWord, Word: nextWord}, nil
}
