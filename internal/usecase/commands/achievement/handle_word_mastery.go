package achievement

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/domain/achievement"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	"github.com/bits-and-blooms/bitset"
)

// NotificationType indicates what kind of message to send the user.
type NotificationType int

const (
	// NotifyUnlock means the user completed an achievement and should be congratulated.
	NotifyUnlock NotificationType = iota
	// NotifyProgress means the user made progress but didn't unlock the achievement.
	NotifyProgress
)

// MasteryNotification is the result for a single achievement tier update.
// The presentation layer will use this to generate the correct message for the user.
type MasteryNotification struct {
	Type             NotificationType
	Achievement      achievement.Achievement
	RecentlyRevealed *bitset.BitSet
}

// HandleWordMasteryCommand is the input for our new use case.
type HandleWordMasteryCommand struct {
	UserID uint
}

// HandleWordMasteryHandler is the use case orchestrator.
type HandleWordMasteryHandler struct {
	logger   logger.Logger
	wordRepo word.Repository
	achRepo  achievement.Repository
}

// NewHandleWordMasteryHandler creates a new handler.
func NewHandleWordMasteryHandler(
	appLogger logger.Logger,
	wordRepo word.Repository,
	achRepo achievement.Repository,
) HandleWordMasteryHandler {
	return HandleWordMasteryHandler{
		logger:   appLogger,
		wordRepo: wordRepo,
		achRepo:  achRepo,
	}
}

// Handle executes the logic to check for and award word mastery achievements.
func (h HandleWordMasteryHandler) Handle(ctx context.Context, cmd HandleWordMasteryCommand) ([]MasteryNotification, error) {
	// 1. Get the user's current total of mastered words.
	masteredCount, err := h.wordRepo.CountMasteredWords(ctx, cmd.UserID)
	if err != nil {
		return nil, fmt.Errorf("could not count mastered words for user %d: %w", cmd.UserID, err)
	}

	// 2. Get all the daily achievements, sorted by the required word count.
	allDailyAchievements, err := h.achRepo.FindAllProgressiveDaily(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch daily achievements: %w", err)
	}

	// 3. Get the user's current progress for all achievements to avoid redundant DB calls.
	userProgress, err := h.achRepo.FindUserAchievements(ctx, cmd.UserID)
	if err != nil {
		return nil, fmt.Errorf("could not fetch user achievement progress: %w", err)
	}

	// Create a map for easy lookup of a user's progress on a specific achievement.
	progressMap := make(map[uint]achievement.UserAchievement)
	for _, p := range userProgress {
		progressMap[p.AchievementID] = p
	}

	var notifications []MasteryNotification
	rand.Seed(time.Now().UnixNano())

	// 4. Iterate through each achievement tier to check the user's status.
	for _, ach := range allDailyAchievements {

		userAch, progressExists := progressMap[ach.ID]

		totalGridItems := ach.TotalItems

		// If user has enough words to unlock this tier...
		if masteredCount >= int(ach.MinWordRequired) {
			// ...and they haven't already completed it...
			if !progressExists || userAch.CompletedAt == nil {
				oldState := bitset.New(totalGridItems)
				if progressExists && userAch.State != nil {
					oldState = userAch.State.Clone()
				}

				now := time.Now()
				if !progressExists {
					userAch = achievement.UserAchievement{UserID: cmd.UserID, AchievementID: ach.ID}
				}
				userAch.State = bitset.New(totalGridItems).SetAll()
				userAch.CompletedAt = &now

				if err := h.achRepo.SaveUserAchievement(ctx, userAch, totalGridItems); err != nil {
					h.logger.Error("failed to save unlocked achievement", "error", err)
					continue
				}
				recentlyRevealed := oldState.SymmetricDifference(userAch.State)
				notifications = append(notifications, MasteryNotification{Type: NotifyUnlock, Achievement: ach, RecentlyRevealed: recentlyRevealed})
			}
		} else { // This is the user's current "active" tier.
			currentRevealedCount := 0
			if progressExists && userAch.State != nil {
				currentRevealedCount = int(userAch.State.Count())
			}

			// If their mastered word count is higher than what's currently shown, update the progress.
			if masteredCount > currentRevealedCount {
				if !progressExists {
					userAch = achievement.UserAchievement{
						UserID:        cmd.UserID,
						AchievementID: ach.ID,
						State:         bitset.New(totalGridItems),
					}
				}

				itemsToReveal := masteredCount - currentRevealedCount
				recentlyRevealed := bitset.New(totalGridItems)

				var availableIndices []uint
				for i := uint(0); i < totalGridItems; i++ {
					if !userAch.State.Test(i) {
						availableIndices = append(availableIndices, i)
					}
				}

				rand.Shuffle(len(availableIndices), func(i, j int) {
					availableIndices[i], availableIndices[j] = availableIndices[j], availableIndices[i]
				})

				revealedCount := 0
				for _, idx := range availableIndices {
					if revealedCount >= itemsToReveal {
						break
					}
					userAch.State.Set(idx)
					recentlyRevealed.Set(idx)
					revealedCount++
				}

				if err := h.achRepo.SaveUserAchievement(ctx, userAch, totalGridItems); err != nil {
					h.logger.Error("failed to save achievement progress", "error", err)
				} else {
					notifications = append(notifications, MasteryNotification{Type: NotifyProgress, Achievement: ach, RecentlyRevealed: recentlyRevealed})
				}
			}
			// Since we found the user's active tier, we don't need to check higher tiers.
			break
		}
	}

	return notifications, nil
}
