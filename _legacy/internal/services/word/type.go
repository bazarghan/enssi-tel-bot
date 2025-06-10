package word

import (
	"errors"
	"time"
)

// --- Service-Specific Errors ---
var (
	ErrWordNotFound           = errors.New("word service: word not found")
	ErrCourseWordLinkNotFound = errors.New("word service: link between course and word not found for this index")
	ErrWordSourceNotFound     = errors.New("word service: detailed source information for the word not found")
	ErrPronunciationNotFound  = errors.New("word service: pronunciation record not found")
	ErrFormattingFailed       = errors.New("word service: failed to format word details")
	ErrFileCacheFailed        = errors.New("word service: failed to cache Telegram file ID")
	ErrMarkStudiedFailed      = errors.New("word service: failed to mark word as studied")
	ErrInvalidInput           = errors.New("word service: invalid input provided")
	ErrWordStudiedNotFound    = errors.New("word service: word studied record not found for user and word")
)

// --- Data Transfer Objects (DTOs) / View Models ---

// WordDisplayData contains all information needed by a handler to display a word to the user.
type WordDisplayData struct {
	WordID             uint
	CourseID           uint // Added for context
	CourseWordIndex    uint // The index within the course
	Title              string
	FormattedText      string // The main textual content (Markdown/HTML)
	TelegramImageID    string // Existing Telegram File ID for the primary image
	TelegramImageDocID string // Existing Telegram File ID for the image as a document
	ImageURL           string // Fallback URL if TelegramImageID is not available (from models.Image)
	Pronunciations     []PronunciationData
	// Add other relevant fields for display
}

// PronunciationData holds information for sending an audio pronunciation.
type PronunciationData struct {
	ID              uint // Pronunciation model ID, useful for caching FileID
	Region          string
	TelegramVoiceID string // Existing Telegram File ID
	AudioURL        string // Fallback URL to download if File ID not available
}

// WordStudiedView is a DTO for words due for review.
// It includes the WordID and any other info needed by QuizService to create a review question.
type WordStudiedView struct {
	UserID             uint
	WordID             uint
	WordTitle          string    // Title of the word for quick reference or display
	NextReviewAt       time.Time // For information or sorting, if needed by caller
	ReviewIntervalDays uint      // Current interval
	// Add other fields if QuizService needs them for question generation for this word.
}
