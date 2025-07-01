package course

import (
	"context"
	"errors"
	achCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/achievement"
	quizCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/quiz"
	"log"

	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
)

// HandleQuizCompletionCommand defines the input for processing a quiz result.
type HandleQuizCompletionCommand struct {
	UserID   uint
	CourseID uint
	Result   quiz.Result
}

// HandleQuizCompletionHandler processes the command.
type HandleQuizCompletionHandler struct {
	logger               logger.Logger
	courseRepo           course.Repository
	wordRepo             word.Repository
	createCourseQuiz     quizCmd.CreateCourseQuizHandler
	awardProgressHandler achCmd.AwardProgressHandler
}

// HandleQuizCompletionResult tells the presentation layer what to do next.
// It reuses the StartSessionResult structure for consistency.
type HandleQuizCompletionResult = StartSessionResult

// NewHandleQuizCompletionHandler creates a new handler.
func NewHandleQuizCompletionHandler(
	appLogger logger.Logger,
	courseRepo course.Repository,
	wordRepo word.Repository,
	createCourseQuiz quizCmd.CreateCourseQuizHandler,
	awardProgressHandler achCmd.AwardProgressHandler,
) HandleQuizCompletionHandler {
	return HandleQuizCompletionHandler{
		logger:               appLogger,
		courseRepo:           courseRepo,
		wordRepo:             wordRepo,
		createCourseQuiz:     createCourseQuiz,
		awardProgressHandler: awardProgressHandler,
	}
}

// Handle executes the command.
func (h HandleQuizCompletionHandler) Handle(ctx context.Context, cmd HandleQuizCompletionCommand) (HandleQuizCompletionResult, error) {
	// This variable will hold the ID of the achievement if it's updated.
	var updatedAchievementID uint

	if cmd.Result.Passed {
		// --- LOGIC FOR A PASSED QUIZ ---

		// 1. Award Achievement Progress
		domainCourse, err := h.courseRepo.FindByID(ctx, cmd.CourseID)
		if err != nil {
			log.Printf("Cannot find course %d to award achievement: %v", cmd.CourseID, err)
		} else if domainCourse.LinkedAchievementID != 0 {
			achCmd := achCmd.AwardProgressCommand{
				UserID:        cmd.UserID,
				AchievementID: domainCourse.LinkedAchievementID,
				ItemsToReveal: 12, // Award 12 "pixels" per passed quiz
			}
			if err := h.awardProgressHandler.Handle(ctx, achCmd); err != nil {
				log.Printf("Failed to award achievement progress: %v", err)
			} else {
				// If awarding was successful, store the ID to be returned later.
				updatedAchievementID = domainCourse.LinkedAchievementID
			}
		}

		// 2. Mark words from the passed block as "studied" in the SRS
		offset := uint(0)
		if cmd.Result.TriggerProgress > WordsPerQuizBlock {
			offset = cmd.Result.TriggerProgress - WordsPerQuizBlock
		}
		wordsInBlock, err := h.wordRepo.FindWordIDsByCourseBlock(ctx, cmd.CourseID, WordsPerQuizBlock, offset)
		if err != nil {
			log.Printf("Could not get words for block to mark as studied: %v", err)
		} else {
			for _, wordID := range wordsInBlock {
				studiedWord, err := h.wordRepo.FindStudiedWord(ctx, cmd.UserID, wordID)
				if err != nil && !errors.Is(err, word.ErrStudiedWordNotFound) {
					log.Printf("error checking for studied word %d: %v", wordID, err)
					continue
				}
				if errors.Is(err, word.ErrStudiedWordNotFound) {
					studiedWord = word.StudiedWord{UserID: cmd.UserID, WordID: wordID}
				}
				studiedWord.CalculateNextReview(true) // 'true' because they passed the quiz
				h.wordRepo.SaveStudiedWord(ctx, studiedWord)
			}
		}

	} else {
		// --- LOGIC FOR A FAILED QUIZ ---
		if cmd.Result.ShouldResetProgress {
			err := h.courseRepo.SetProgress(ctx, cmd.UserID, cmd.CourseID, cmd.Result.SuggestedNewProgress)
			if err != nil {
				log.Printf("Failed to reset progress for user %d in course %d: %v", cmd.UserID, cmd.CourseID, err)
			} else {
				log.Printf("Successfully reset progress for user %d in course %d to %d", cmd.UserID, cmd.CourseID, cmd.Result.SuggestedNewProgress)
			}
		}
	}

	// 3. After handling the quiz consequences, determine the next step in the course.

	startSessionHandler := NewStartSessionHandler(h.logger, h.courseRepo, h.wordRepo, h.createCourseQuiz)
	result, err := startSessionHandler.Handle(ctx, StartSessionCommand{UserID: cmd.UserID, CourseID: cmd.CourseID})

	if err != nil {
		return HandleQuizCompletionResult{}, err
	}

	// 4. IMPORTANT: Add the achievement ID to the final result before returning.
	result.UpdatedAchievementID = updatedAchievementID

	return result, nil
}
