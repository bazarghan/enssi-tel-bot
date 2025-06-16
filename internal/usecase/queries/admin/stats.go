package admin

import (
	"context"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
)

// Stats is the result DTO for the use case.
type Stats struct {
	TotalUsers int64
}

// GetStatsHandler processes the query to get bot statistics.
type GetStatsHandler struct {
	userRepo user.Repository
	// You could add other repos here later, like courseRepo to count courses
}

func NewGetStatsHandler(userRepo user.Repository) GetStatsHandler {
	return GetStatsHandler{userRepo: userRepo}
}

func (h GetStatsHandler) Handle(ctx context.Context) (Stats, error) {
	totalUsers, err := h.userRepo.CountTotalUsers(ctx)
	if err != nil {
		return Stats{}, err
	}

	return Stats{TotalUsers: totalUsers}, nil
}
