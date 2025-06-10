package course

import (
	"context"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"log"
)

// ListCoursesQuery defines the input for listing courses for a specific user.
type ListCoursesQuery struct {
	UserID uint
}

// CourseSummary is a DTO for the use case layer representing a course in a list.
type CourseSummary struct {
	ID                 uint
	PersianTitle       string
	ProgressPercentage int
	IsCompleted        bool
}

// ListCoursesHandler processes the query to list all courses.
type ListCoursesHandler struct {
	courseRepo course.Repository
}

// NewListCoursesHandler creates a new handler.
func NewListCoursesHandler(courseRepo course.Repository) ListCoursesHandler {
	return ListCoursesHandler{courseRepo: courseRepo}
}

// Handle executes the query.
func (h ListCoursesHandler) Handle(ctx context.Context, q ListCoursesQuery) ([]CourseSummary, error) {
	courses, err := h.courseRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not find all courses: %w", err)
	}

	summaries := make([]CourseSummary, 0, len(courses))
	for _, c := range courses {
		// Not finding progress for a course is not a fatal error for the whole list.
		progress, err := h.courseRepo.GetUserProgress(ctx, q.UserID, c.ID)
		if err != nil {
			log.Printf("could not get user progress for course %d: %v\n", c.ID, err)
		}

		summaries = append(summaries, CourseSummary{
			ID:                 c.ID,
			PersianTitle:       c.PersianTitle,
			ProgressPercentage: progress.ProgressPercentage,
			IsCompleted:        progress.IsCompleted,
		})
	}

	return summaries, nil
}
