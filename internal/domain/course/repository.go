package course

import "context"

// Repository defines the persistence port for course-related data.
type Repository interface {
	// FindAll retrieves all available courses.
	FindAll(ctx context.Context) ([]Course, error)

	// FindByID retrieves a single course by its ID.
	FindByID(ctx context.Context, courseID uint) (Course, error)

	// FindByPersianTitle retrieves a single course by its Persian title.
	FindByPersianTitle(ctx context.Context, title string) (Course, error)

	// GetUserProgress retrieves a specific user's progress for a given course.
	GetUserProgress(ctx context.Context, userID uint, courseID uint) (UserProgress, error)

	// GetTotalWords retrieves the number of words in a course.
	GetTotalWords(ctx context.Context, courseID uint) (int, error)

	// GetOrCreateUserCourse ensures a user is enrolled in a course, creating a progress record if needed.
	GetOrCreateUserCourse(ctx context.Context, userID uint, courseID uint) (UserProgress, error)
}
