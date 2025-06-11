package word

import (
	"context"
	"time"
)

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

	// GetWordsDueForReview retrieves all words for a user that are due for review.
	GetWordsDueForReview(ctx context.Context, userID uint, now time.Time) ([]StudiedWord, error)

	// FindDisplayableWordByIndex fetches a word and its associated media FileIDs.
	FindDisplayableWordByIndex(ctx context.Context, courseID uint, index uint) (DisplayableWord, error)

	// CacheImageFileIDs updates a CourseWord entry with Telegram File IDs for an image.
	CacheImageFileIDs(ctx context.Context, courseWordID uint, imageFileID, imageDocFileID string) error

	// CacheVoiceFileID updates a Pronunciation entry with a Telegram File ID for a voice message.
	CacheVoiceFileID(ctx context.Context, pronunciationID uint, voiceFileID string) error
}
