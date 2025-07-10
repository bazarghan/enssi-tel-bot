package achievement

import (
	"context"
	"errors"
	"fmt"
	"github.com/bazarghan/enssi-tel-bot/internal/domain/achievement"
	"github.com/bits-and-blooms/bitset"
	"math/rand"
	"time"
)

// AwardProgressCommand defines the input for updating achievement progress.
type AwardProgressCommand struct {
	UserID        uint
	AchievementID uint
	ItemsToReveal int
}

// AwardProgressHandler processes the command.
type AwardProgressHandler struct {
	repo achievement.Repository
}

// NewAwardProgressHandler creates a new handler.
func NewAwardProgressHandler(repo achievement.Repository) AwardProgressHandler {
	return AwardProgressHandler{repo: repo}
}

// Handle executes the command.
func (h AwardProgressHandler) Handle(ctx context.Context, cmd AwardProgressCommand) (*bitset.BitSet, error) {
	ach, err := h.repo.FindByID(ctx, cmd.AchievementID)
	if err != nil {
		return nil, fmt.Errorf("could not find achievement to award progress: %w", err)
	}

	userAch, err := h.repo.GetUserAchievement(ctx, cmd.UserID, cmd.AchievementID)
	if err != nil {
		if errors.Is(err, achievement.ErrUserAchNotFound) {
			// First time progress is awarded, create a new record.
			userAch = achievement.UserAchievement{
				UserID:        cmd.UserID,
				AchievementID: cmd.AchievementID,
				State:         bitset.New(ach.TotalItems),
			}
		} else {
			return nil, fmt.Errorf("could not get user achievement progress: %w", err)
		}
	}

	// This will store the pixels we are about to reveal.
	recentlyRevealed := bitset.New(ach.TotalItems)

	// Find all available (unset) indices.
	var availableIndices []uint
	for i := uint(0); i < ach.TotalItems; i++ {
		if !userAch.State.Test(i) {
			availableIndices = append(availableIndices, i)
		}
	}

	if len(availableIndices) > 0 {
		// Shuffle the list of available indices.
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(availableIndices), func(i, j int) {
			availableIndices[i], availableIndices[j] = availableIndices[j], availableIndices[i]
		})

		// Reveal the first 'itemsToReveal' from the shuffled list.
		revealedCount := 0
		for _, idx := range availableIndices {
			if revealedCount >= cmd.ItemsToReveal {
				break
			}
			userAch.State.Set(idx)
			recentlyRevealed.Set(idx) // Also track it in our new bitset.
			revealedCount++
		}
	}

	err = h.repo.SaveUserAchievement(ctx, userAch, ach.TotalItems)
	if err != nil {
		return nil, err
	}

	return recentlyRevealed, nil
}
