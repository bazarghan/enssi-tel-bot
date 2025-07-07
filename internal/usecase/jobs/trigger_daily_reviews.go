package jobs

import (
	"context"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/notification"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"

	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"

	quizCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/quiz"
	"time"
)

// TriggerDailyReviewsJob is a use case representing the entire daily job.
type TriggerDailyReviewsJob struct {
	logger           logger.Logger
	userRepo         user.Repository
	wordRepo         word.Repository
	quizRepo         quiz.Repository
	createReviewQuiz quizCmd.CreateReviewQuizHandler
	notifier         notification.Notifier
}

// NewTriggerDailyReviewsJob creates a new job handler.
func NewTriggerDailyReviewsJob(
	appLogger logger.Logger,
	userRepo user.Repository,
	wordRepo word.Repository,
	quizRepo quiz.Repository,
	createReviewQuiz quizCmd.CreateReviewQuizHandler,
	notifier notification.Notifier,
) *TriggerDailyReviewsJob {
	return &TriggerDailyReviewsJob{
		logger:           appLogger,
		userRepo:         userRepo,
		wordRepo:         wordRepo,
		quizRepo:         quizRepo,
		createReviewQuiz: createReviewQuiz,
		notifier:         notifier,
	}
}

// Run executes the job logic.
func (j *TriggerDailyReviewsJob) Run() {
	j.logger.Info("Starting daily review check job...")
	ctx := context.Background()

	userIDs, err := j.userRepo.FindAllIDs(ctx)
	if err != nil {
		j.logger.Error("Could not fetch users for daily review job", "error", err)
		return
	}

	for _, userID := range userIDs {
		// In a real high-volume system, this would be parallelized with worker pools.
		j.processUserForReview(ctx, userID)
	}

	j.logger.Info("Daily review check job finished.")
}

func (j *TriggerDailyReviewsJob) processUserForReview(ctx context.Context, userID uint) {
	domainUser, err := j.userRepo.FindByID(ctx, userID)
	if err != nil {
		j.logger.Error("Could not get user details", "userID", userID, "error", err)
		return
	}

	// 1. Check if the user already completed a review today.
	now := time.Now()
	if !domainUser.LastReviewSessionCompletedAt.IsZero() {
		lastReviewDate := domainUser.LastReviewSessionCompletedAt.In(now.Location())
		currentDate := now.In(now.Location())
		if lastReviewDate.Year() == currentDate.Year() && lastReviewDate.YearDay() == currentDate.YearDay() {
			return // Skip user, review already done today.
		}
	}

	// 2. Check if there are any words due for review.
	wordsDue, err := j.wordRepo.GetWordsDueForReview(ctx, userID, now)
	if err != nil {
		j.logger.Error("Could not get words due for review for user", "userID", userID, "error", err)
		return
	}

	if len(wordsDue) == 0 {
		return // No words to review, so do nothing.
	}

	// 4. Notify the user with a message and the main menu keyboard.
	// This forces the main menu to appear on their device.
	message := fmt.Sprintf("👋 سلام! %d کلمه برای مرور روزانه شما آماده است.", len(wordsDue))
	if err := j.notifier.NotifyWithMainMenu(userID, message); err != nil {
		j.logger.Error("Failed to send main menu notification to user", "userID", userID, "error", err)
	}
}
