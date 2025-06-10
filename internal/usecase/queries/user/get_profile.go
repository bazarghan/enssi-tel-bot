package user

import (
	"context"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
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
}

// GetProfileHandler processes the query.
type GetProfileHandler struct {
	userRepo user.Repository
	// Dependencies on other repositories (WordStudied, UserCourse) will be added in later slices.
}

// NewGetProfileHandler creates a new handler.
func NewGetProfileHandler(userRepo user.Repository) GetProfileHandler {
	return GetProfileHandler{userRepo: userRepo}
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

	return GetProfileResult{
		UserID:        domainUser.ID,
		Username:      domainUser.Profile.Username,
		FirstName:     domainUser.Profile.FirstName,
		LastName:      domainUser.Profile.LastName,
		Score:         domainUser.Profile.Score,
		WordsStudied:  0, // Placeholder for this slice
		CoursesActive: 0, // Placeholder for this slice
	}, nil
}

