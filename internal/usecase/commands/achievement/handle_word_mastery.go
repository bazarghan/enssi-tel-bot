package achievement

import (
	"context"
	"fmt"
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
	Type        NotificationType
	Achievement achievement.Achievement
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

	// 4. Iterate through each achievement tier to check the user's status.
	for _, ach := range allDailyAchievements {

		userAch, progressExists := progressMap[ach.ID]

		// --- START OF CHANGE ---
		// Defensively calculate the total grid size from width and height
		// instead of relying on the `total_items` field from the database.
		totalGridItems := ach.GridWidth * ach.GridHeight
		if totalGridItems == 0 {
			// As a fallback, use the DB value if grid dimensions aren't set.
			totalGridItems = ach.TotalItems
		}
		// --- END OF CHANGE ---

		// If user has enough words to unlock this tier...
		if masteredCount >= int(ach.MinWordRequired) {
			// ...and they haven't already completed it...
			if !progressExists || userAch.CompletedAt == nil {
				// Mark as complete and create an "Unlock" notification.
				now := time.Now()
				if !progressExists {
					userAch = achievement.UserAchievement{UserID: cmd.UserID, AchievementID: ach.ID}
				}
				userAch.State = bitset.New(ach.TotalItems).SetAll() // Fill the image completely.
				userAch.CompletedAt = &now

				if err := h.achRepo.SaveUserAchievement(ctx, userAch, totalGridItems); err != nil {
					h.logger.Error("failed to save unlocked achievement", "error", err)
					continue // Move to next achievement
				}
				notifications = append(notifications, MasteryNotification{Type: NotifyUnlock, Achievement: ach})
			}
		} else { // If user does NOT have enough words for this tier...
			// This is their current "active" tier. We update their progress here.

			// Calculate how many "pixels" (revealed items) they should have.
			currentRevealed := 0
			if progressExists && userAch.State != nil {
				currentRevealed = int(userAch.State.Count())
			}

			// If their mastered word count is higher than what's shown, update it.
			if masteredCount > currentRevealed {
				if !progressExists {
					userAch = achievement.UserAchievement{
						UserID:        cmd.UserID,
						AchievementID: ach.ID,
						State:         bitset.New(ach.TotalItems),
					}
				}

				// Reveal pixels up to the mastered word count.
				for i := uint(0); i < uint(masteredCount); i++ {
					userAch.State.Set(i)
				}

				if err := h.achRepo.SaveUserAchievement(ctx, userAch, totalGridItems); err != nil {
					h.logger.Error("failed to save achievement progress", "error", err)
				} else {
					// Add a "Progress" notification only if we actually updated something.
					notifications = append(notifications, MasteryNotification{Type: NotifyProgress, Achievement: ach})
				}
			}

			// Since we found the user's active tier, we don't need to check higher tiers.
			break
		}
	}

	return notifications, nil
}
