package course

import (
	"context"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	"log"
)

// GetOverviewQuery defines the input for getting a course overview.
type GetOverviewQuery struct {
	UserID   uint
	CourseID uint
}

// OverviewResult is a DTO for the use case layer representing a course's details.
type OverviewResult struct {
	ID                     uint
	Title                  string
	PersianTitle           string
	PersianFullDescription string
	TotalWords             int
	ProgressPercentage     int
	IsCompleted            bool
	IsStarted              bool
}

// GetOverviewHandler processes the query to get a course overview.
type GetOverviewHandler struct {
	logger     logger.Logger
	courseRepo course.Repository
}

// NewGetOverviewHandler creates a new handler.
func NewGetOverviewHandler(appLogger logger.Logger, courseRepo course.Repository) GetOverviewHandler {
	return GetOverviewHandler{logger: appLogger, courseRepo: courseRepo}
}

// Handle executes the query.
func (h GetOverviewHandler) Handle(ctx context.Context, q GetOverviewQuery) (OverviewResult, error) {
	c, err := h.courseRepo.FindByID(ctx, q.CourseID)
	if err != nil {
		return OverviewResult{}, fmt.Errorf("could not find course by id %d: %w", q.CourseID, err)
	}

	progress, err := h.courseRepo.GetUserProgress(ctx, q.UserID, c.ID)
	if err != nil {
		log.Printf("could not get user progress for course overview %d: %v\n", c.ID, err)
	}

	return OverviewResult{
		ID:                     c.ID,
		Title:                  c.Title,
		PersianTitle:           c.PersianTitle,
		PersianFullDescription: c.PersianDescription,
		TotalWords:             c.TotalWords,
		ProgressPercentage:     progress.ProgressPercentage,
		IsCompleted:            progress.IsCompleted,
		IsStarted:              progress.IsStarted,
	}, nil
}
