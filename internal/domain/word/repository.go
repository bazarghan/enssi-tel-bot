package word

import "context"

// Repository defines the port for word persistence operations.
type Repository interface {
	// FindByCourseIndex finds a single word associated with a course at a specific index.
	FindByCourseIndex(ctx context.Context, courseID uint, index uint) (Word, error)

	// FindStudiedWord retrieves a user's SRS data for a specific word.
	FindStudiedWord(ctx context.Context, userID, wordID uint) (StudiedWord, error)

	// SaveStudiedWord creates or updates a user's SRS data for a word.
	SaveStudiedWord(ctx context.Context, sw StudiedWord) error

	// FindWordIDsByCourseBlock retrieves a slice of word IDs for a specific block in a course.
	FindWordIDsByCourseBlock(ctx context.Context, courseID uint, limit uint, offset uint) ([]uint, error)
}

