package course

import (
	"context"
	"errors"
	"log"

	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
)

// HandleQuizCompletionCommand defines the input for processing a quiz result.
type HandleQuizCompletionCommand struct {
	UserID   uint
	CourseID uint
	Result   quiz.Result
}

// HandleQuizCompletionResult tells the presentation layer what to do next.
// It reuses the StartSessionResult structure for consistency.
type HandleQuizCompletionResult = StartSessionResult

// HandleQuizCompletionHandler processes the command.
type HandleQuizCompletionHandler struct {
	courseRepo course.Repository
	wordRepo   word.Repository
}

// NewHandleQuizCompletionHandler creates a new handler.
func NewHandleQuizCompletionHandler(courseRepo course.Repository, wordRepo word.Repository) HandleQuizCompletionHandler {
	return HandleQuizCompletionHandler{courseRepo: courseRepo, wordRepo: wordRepo}
}

// Handle executes the command.
func (h HandleQuizCompletionHandler) Handle(ctx context.Context, cmd HandleQuizCompletionCommand) (HandleQuizCompletionResult, error) {
	if cmd.Result.Passed {
		// Mark the words from the quiz block as studied.
		// In a real system, you'd fetch the words associated with the quiz.
		// For now, we assume the trigger progress marks the end of the block.
		offset := cmd.Result.SuggestedNewProgress // On pass, this is the start of the block
		if cmd.Result.TriggerProgress > WordsPerQuizBlock {
			offset = cmd.Result.TriggerProgress - WordsPerQuizBlock
		}

		wordsInBlock, err := h.wordRepo.FindWordIDsByCourseBlock(ctx, cmd.CourseID, WordsPerQuizBlock, offset)
		if err != nil {
			log.Printf("Could not get words for block to mark as studied: %v", err)
		} else {
			for _, wordID := range wordsInBlock {
				// Mark as studied using the SRS primitive.
				studiedWord, err := h.wordRepo.FindStudiedWord(ctx, cmd.UserID, wordID)
				if err != nil && !errors.Is(err, word.ErrStudiedWordNotFound) {
					log.Printf("error checking for studied word %d: %v", wordID, err)
					continue
				}
				if errors.Is(err, word.ErrStudiedWordNotFound) {
					studiedWord = word.StudiedWord{UserID: cmd.UserID, WordID: wordID}
				}
				studiedWord.CalculateNextReview(true)
				h.wordRepo.SaveStudiedWord(ctx, studiedWord)
			}
		}
	} else {
		// If failed, reset progress.
		if cmd.Result.ShouldResetProgress {
			// This requires a new method in the repository.
			// For now, we log it. A future slice would add `SetProgress`.
			log.Printf("TODO: Resetting progress for user %d in course %d to %d", cmd.UserID, cmd.CourseID, cmd.Result.SuggestedNewProgress)
		}
	}

	// After handling the result, determine the next step by re-evaluating the session.
	startSessionHandler := NewStartSessionHandler(h.courseRepo, h.wordRepo)
	return startSessionHandler.Handle(ctx, StartSessionCommand{UserID: cmd.UserID, CourseID: cmd.CourseID})
}
