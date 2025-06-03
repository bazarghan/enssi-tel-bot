package word

// WordService defines the interface for word-related operations.
type WordService interface {
	// GetWordDetailsForCourse retrieves and prepares a specific word from a course for display.
	// courseWordIndex is the 'Index' from the CourseWord model.
	GetWordDetailsForCourse(courseID uint, courseWordIndex uint, userID uint) (*WordDisplayData, error)

	// CacheTelegramFileIDForPronunciation updates a Pronunciation model with a Telegram File ID.
	CacheTelegramFileIDForPronunciation(pronunciationID uint, telegramFileID string) error

	// CacheTelegramFileIDForCourseWordImage updates a CourseWord model with Telegram File IDs for its image.
	CacheTelegramFileIDForCourseWordImage(courseWordID uint, imageFileID string, imageDocFileID string) error // Assuming CourseWord has a direct ID

	// MarkWordAsStudied records that a user has studied a specific word in a course context.
	MarkWordAsStudied(userID uint, wordID uint, courseID uint) error
}

