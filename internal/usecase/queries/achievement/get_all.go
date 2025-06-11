package achievement

import (
	"context"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/achievement"
)

// GetAllResult is a DTO for the use case layer.
type GetAllResult struct {
	ID          uint
	Title       string
	Description string
}

// GetAllHandler processes the query to list all defined achievements.
type GetAllHandler struct {
	repo achievement.Repository
}

// NewGetAllHandler creates a new handler.
func NewGetAllHandler(repo achievement.Repository) GetAllHandler {
	return GetAllHandler{repo: repo}
}

// Handle executes the query.
func (h GetAllHandler) Handle(ctx context.Context) ([]GetAllResult, error) {
	allAchievements, err := h.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]GetAllResult, len(allAchievements))
	for i, ach := range allAchievements {
		results[i] = GetAllResult{
			ID:          ach.ID,
			Title:       ach.Title,
			Description: ach.Description,
		}
	}
	return results, nil
}
