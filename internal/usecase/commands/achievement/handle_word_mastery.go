package achievement

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/bazarghan/enssi-tel-bot/internal/domain/achievement"
	"github.com/bazarghan/enssi-tel-bot/internal/domain/word"
	"github.com/bazarghan/enssi-tel-bot/internal/platform/observability/logger"
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

	// This flag tells the loop when to stop SENDING notifications, but not when to stop UPDATING.
	var stopNotifying = false

	// 4. Loop through every achievement tier without breaking.
	for _, ach := range allDailyAchievements {
		userAch, progressExists := progressMap[ach.ID]

		// If the achievement was already fully completed in a previous session, do nothing.
		if progressExists && userAch.CompletedAt != nil {
			continue
		}

		// --- A: Logic for UNLOCKING a tier ---
		if masteredCount >= int(ach.MinWordRequired) {
			now := time.Now()
			if !progressExists {
				userAch = achievement.UserAchievement{UserID: cmd.UserID, AchievementID: ach.ID}
			}
			userAch.State = bitset.New(ach.TotalItems).SetAll()
			userAch.CompletedAt = &now

			if err := h.achRepo.SaveUserAchievement(ctx, userAch, ach.TotalItems); err != nil {
				h.logger.Error("failed to save unlocked achievement", "error", err)
				continue
			}

			// If the notification switch is still off, send the unlock message.
			if !stopNotifying {
				notifications = append(notifications, MasteryNotification{Type: NotifyUnlock, Achievement: ach, RecentlyRevealed: nil})
			}

			// --- B: Logic for PROGRESS and SILENT UPDATES ---
		} else {
			oldState := bitset.New(ach.TotalItems)
			if progressExists && userAch.State != nil {
				oldState = userAch.State.Clone()
			}

			currentRevealedCount := int(oldState.Count())

			if masteredCount > currentRevealedCount {
				// This block runs for ANY achievement that needs a progress update,
				// both the first one and all subsequent silent ones.
				if !progressExists {
					userAch = achievement.UserAchievement{
						UserID:        cmd.UserID,
						AchievementID: ach.ID,
						State:         bitset.New(ach.TotalItems),
					}
				}

				itemsToReveal := masteredCount - currentRevealedCount
				newlyRevealed := bitset.New(ach.TotalItems)

				var availableIndices []uint
				for i := uint(0); i < ach.TotalItems; i++ {
					if !userAch.State.Test(i) {
						availableIndices = append(availableIndices, i)
					}
				}
				rand.Shuffle(len(availableIndices), func(i, j int) { availableIndices[i], availableIndices[j] = availableIndices[j], availableIndices[i] })

				revealedCount := 0
				for _, idx := range availableIndices {
					if revealedCount >= itemsToReveal {
						break
					}
					userAch.State.Set(idx)
					newlyRevealed.Set(idx)
					revealedCount++
				}

				if err := h.achRepo.SaveUserAchievement(ctx, userAch, ach.TotalItems); err != nil {
					h.logger.Error("failed to save achievement progress", "error", err)
					continue
				}

				// If the notification switch is off, this is the FIRST progress update.
				// Send the notification AND turn the switch on.
				if !stopNotifying {
					notifications = append(notifications, MasteryNotification{Type: NotifyProgress, Achievement: ach, RecentlyRevealed: newlyRevealed})
					stopNotifying = true
				}
			}
		}
	}

	return notifications, nil
}
