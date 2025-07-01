package user

import (
	"context"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/achievement"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	"log"
	"time"
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
	logger   logger.Logger
	userRepo user.Repository
	achRepo  achievement.Repository
	// Dependencies on other repositories (WordStudied, UserCourse) will be added in later slices.
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
) GetProfileHandler {
	return GetProfileHandler{
		logger:   appLogger,
		userRepo: userRepo,
		achRepo:  achRepo,
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

	// TODO: In future slices, call other repositories to get WordsStudied and CoursesActive counts.

	userAchievements, err := h.achRepo.FindUserAchievements(ctx, q.UserID)
	if err != nil {
		log.Printf("Could not fetch achievements for user %d: %v", q.UserID, err)
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
		WordsStudied:  0, // Placeholder for this slice
		CoursesActive: 0, // Placeholder for this slice
		Achievements:  achResults,
	}, nil
}
