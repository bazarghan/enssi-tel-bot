package word

import "context"

// Repository defines the port for word persistence operations.
type Repository interface {
	// FindByCourseIndex finds a single word associated with a course at a specific index.
	FindByCourseIndex(ctx context.Context, courseID uint, index uint) (Word, error)
}
