package word

import "time"

// WordService defines the interface for word-related operations.
type WordService interface {
	// GetWordDetailsForCourse retrieves and prepares a specific word from a course for display.
	// courseWordIndex is the 'Index' from the CourseWord model.
	GetWordDetailsForCourse(courseID uint, courseWordIndex uint, userID uint) (*WordDisplayData, error)

	// CacheTelegramFileIDForPronunciation updates a Pronunciation model with a Telegram File ID.
	CacheTelegramFileIDForPronunciation(pronunciationID uint, telegramFileID string) error

	// CacheTelegramFileIDForCourseWordImage updates a CourseWord model with Telegram File IDs for its image.
	CacheTelegramFileIDForCourseWordImage(courseWordID uint, imageFileID string, imageDocFileID string) error

	MarkWordAsStudied(userID uint, wordID uint, courseID uint) error

	// UpdateWordReviewSchedule updates the spaced repetition schedule for a word based on review performance.
	UpdateWordReviewSchedule(userID uint, wordID uint, wasCorrect bool) error

	// GetWordsDueForReview retrieves all WordStudied records for a user that are due for review by 'now'.
	GetWordsDueForReview(userID uint, now time.Time) ([]WordStudiedView, error) // Changed return type
}
