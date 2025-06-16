package review

import (
	"context"
	"log"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
)

// Handler processes the HasPendingReviewQuery.
type Handler struct {
	userRepo user.Repository
	wordRepo word.Repository
}

// NewHandler creates a new handler.
func NewHandler(userRepo user.Repository, wordRepo word.Repository) Handler {
	return Handler{userRepo: userRepo, wordRepo: wordRepo}
}

// Handle executes the query. It returns true if a user is eligible for a daily review.
func (h Handler) Handle(ctx context.Context, q HasPendingReviewQuery) (bool, error) {
	// 1. First, check if the user has already completed a review session today.
	user, err := h.userRepo.FindByID(ctx, q.UserID)
	if err != nil {
		return false, err // Return false if we can't find the user
	}

	now := time.Now()
	lastReview := user.LastReviewSessionCompletedAt
	if !lastReview.IsZero() && lastReview.Year() == now.Year() && lastReview.YearDay() == now.YearDay() {
		// User has already reviewed today. They have no pending review.
		return false, nil
	}

	// 2. If they haven't reviewed today, check if they have any words waiting.
	wordsDue, err := h.wordRepo.GetWordsDueForReview(ctx, q.UserID, now)
	if err != nil {
		log.Printf("Could not check for due words for user %d: %v", q.UserID, err)
		return false, err
	}

	// If there are words due, they have a "pending review".
	return len(wordsDue) > 0, nil
}
