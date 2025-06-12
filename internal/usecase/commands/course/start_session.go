package course

import (
	"context"
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
	quizCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/quiz"
	"log"
)

type Step int

const WordsPerQuizBlock = 12
const (
	ShowWord Step = iota
	ShowQuiz
	CourseEnded
)

// PronunciationDisplayData is a DTO for the use case layer.
type PronunciationDisplayData struct {
	ID              uint
	Region          string
	AudioURL        string
	TelegramVoiceID string
}

// WordDisplayData is a DTO for the use case layer.
type WordDisplayData struct {
	CourseWordID       uint
	Title              string
	FormattedText      string
	ImageURL           string
	TelegramImageID    string
	TelegramImageDocID string
	Pronunciations     []PronunciationDisplayData
}

// StartSessionCommand defines the input for starting a course session.
type StartSessionCommand struct {
	UserID   uint
	CourseID uint
}

// StartSessionResult tells the presentation layer what to show next.
type StartSessionResult struct {
	NextStep      Step
	Word          WordDisplayData
	MessageToUser string

	QuizAttempt quiz.Attempt
}

// StartSessionHandler processes the command.
type StartSessionHandler struct {
	courseRepo       course.Repository
	wordRepo         word.Repository
	createCourseQuiz quizCmd.CreateCourseQuizHandler
}

// NewStartSessionHandler creates a new handler.
func NewStartSessionHandler(
	courseRepo course.Repository,
	wordRepo word.Repository,
	createCourseQuiz quizCmd.CreateCourseQuizHandler,
) StartSessionHandler {
	return StartSessionHandler{
		courseRepo:       courseRepo,
		wordRepo:         wordRepo,
		createCourseQuiz: createCourseQuiz,
	}
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

	// Check if a quiz is due for the current progress point.
	if progress.WordsCompleted > 0 && progress.WordsCompleted%WordsPerQuizBlock == 0 {
		quizCmd := quizCmd.CreateCourseQuizCommand{
			UserID:          cmd.UserID,
			CourseID:        cmd.CourseID,
			TriggerProgress: uint(progress.WordsCompleted),
		}
		quizResult, err := h.createCourseQuiz.Handle(ctx, quizCmd)
		if err != nil && !errors.Is(err, quiz.ErrAttemptAlreadyCompleted) {
			log.Printf("Failed to create or find quiz for user %d, course %d: %v", cmd.UserID, cmd.CourseID, err)
		} else if quizResult.QuizAttempt.ID != 0 && !quizResult.QuizAttempt.IsCompleted {
			return StartSessionResult{
				NextStep:    ShowQuiz,
				QuizAttempt: quizResult.QuizAttempt,
			}, nil
		}
	}

	nextWordIndex := uint(progress.WordsCompleted + 1)
	displayableWord, err := h.wordRepo.FindDisplayableWordByIndex(ctx, cmd.CourseID, nextWordIndex)

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
		Word:     mapToWordDisplayData(displayableWord),
	}, nil

}

// mapToWordDisplayData converts the repository DTO to the use case DTO.
// In a real app, this might use a library like `automapper`.
func mapToWordDisplayData(repoWord word.DisplayableWord) WordDisplayData {
	// Here, you would call a formatter to create the Markdown text
	formattedText := "" // Placeholder, formatter will be called in handler

	prons := make([]PronunciationDisplayData, len(repoWord.Pronunciations))
	for i, p := range repoWord.Pronunciations {
		prons[i] = PronunciationDisplayData{
			ID:              p.ID,
			Region:          p.Region,
			AudioURL:        p.AudioURL,
			TelegramVoiceID: p.TelegramVoiceID,
		}
	}

	return WordDisplayData{
		CourseWordID:       repoWord.CourseWordID,
		Title:              repoWord.Title,
		FormattedText:      formattedText,
		ImageURL:           repoWord.ImageURL,
		TelegramImageID:    repoWord.TelegramImageID,
		TelegramImageDocID: repoWord.TelegramImageDocID,
		Pronunciations:     prons,
	}
}
