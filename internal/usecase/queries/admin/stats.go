package admin

import (
	"context"

	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
)

// Stats is the result DTO for the use case.
type Stats struct {
	TotalUsers        int64
	TotalWordsStudied int64
}

// GetStatsHandler processes the query to get bot statistics.
type GetStatsHandler struct {
	logger   logger.Logger
	userRepo user.Repository
	wordRepo word.Repository // <-- ADD THIS FIELD
	// You could add other repos here later, like courseRepo to count courses
}

func NewGetStatsHandler(
	appLogger logger.Logger,
	userRepo user.Repository,
	wordRepo word.Repository,
) GetStatsHandler {
	return GetStatsHandler{logger: appLogger, userRepo: userRepo, wordRepo: wordRepo}
}

func (h GetStatsHandler) Handle(ctx context.Context) (Stats, error) {
	totalUsers, err := h.userRepo.CountTotalUsers(ctx)
	if err != nil {
		return Stats{}, err
	}

	totalWordsStudied, err := h.wordRepo.CountTotalStudiedWords(ctx)
	if err != nil {
		return Stats{}, err
	}

	return Stats{
		TotalUsers:        totalUsers,
		TotalWordsStudied: totalWordsStudied,
	}, nil
}
