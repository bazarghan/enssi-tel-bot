package user

import (
	"context"
	"fmt"
	"time"

	"github.com/bazarghan/enssi-tel-bot/internal/domain/achievement"
	"github.com/bazarghan/enssi-tel-bot/internal/domain/course"
	"github.com/bazarghan/enssi-tel-bot/internal/domain/user"
	"github.com/bazarghan/enssi-tel-bot/internal/domain/word"
	"github.com/bazarghan/enssi-tel-bot/internal/platform/observability/logger"
)

// GetProfileQuery defines the input for the query.
type GetProfileQuery struct {
	UserID uint
}

// GetProfileResult is a DTO for the user profile view.
type GetProfileResult struct {
	UserID        uint
	Username      string
	FirstName     string
	LastName      string
	Score         uint
	WordsStudied  int // Note: Will be implemented in a later slice
	CoursesActive int // Note: Will be implemented in a later slice

	Achievements []AchievementResult
}

// GetProfileHandler processes the query.
type GetProfileHandler struct {
	logger     logger.Logger
	userRepo   user.Repository
	achRepo    achievement.Repository
	wordRepo   word.Repository   // Added
	courseRepo course.Repository // Added
}

// AchievementResult is a DTO for achievements included in the profile.
type AchievementResult struct {
	Title    string
	EarnedOn time.Time
}

// NewGetProfileHandler creates a new handler.
func NewGetProfileHandler(
	appLogger logger.Logger,
	userRepo user.Repository,
	achRepo achievement.Repository,
	wordRepo word.Repository, // Added
	courseRepo course.Repository, // Added
) GetProfileHandler {
	return GetProfileHandler{
		logger:     appLogger,
		userRepo:   userRepo,
		achRepo:    achRepo,
		wordRepo:   wordRepo,
		courseRepo: courseRepo,
	}
}

// Handle executes the query.
func (h GetProfileHandler) Handle(ctx context.Context, q GetProfileQuery) (GetProfileResult, error) {
	if q.UserID == 0 {
		return GetProfileResult{}, user.ErrInvalidInput
	}

	domainUser, err := h.userRepo.FindByID(ctx, q.UserID)
	if err != nil {
		return GetProfileResult{}, fmt.Errorf("failed to find user for profile: %w", err)
	}

	wordsStudied, err := h.wordRepo.CountStudiedWords(ctx, q.UserID)
	if err != nil {
		h.logger.Warn("Could not fetch words studied count", "userID", q.UserID, "error", err)
		// Default to 0 on error
	}

	coursesActive, err := h.courseRepo.CountActive(ctx, q.UserID)
	if err != nil {
		h.logger.Warn("Could not fetch active courses count", "userID", q.UserID, "error", err)
		// Default to 0 on error
	}

	userAchievements, err := h.achRepo.FindUserAchievements(ctx, q.UserID)
	if err != nil {
		h.logger.Warn("Could not fetch achievements for user", "userID", q.UserID, "error", err)
		// Non-fatal error, we can still show the rest of the profile
	}

	achResults := make([]AchievementResult, len(userAchievements))
	for i, ua := range userAchievements {
		achResults[i] = AchievementResult{
			Title:    ua.Details.Title,
			EarnedOn: ua.EarnedAt,
		}
	}

	return GetProfileResult{
		UserID:        domainUser.ID,
		Username:      domainUser.Profile.Username,
		FirstName:     domainUser.Profile.FirstName,
		LastName:      domainUser.Profile.LastName,
		Score:         domainUser.Profile.Score,
		WordsStudied:  wordsStudied,
		CoursesActive: coursesActive,
		Achievements:  achResults,
	}, nil
}
