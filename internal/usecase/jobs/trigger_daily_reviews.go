package jobs

import (
	"context"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/notification"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"

	quizCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/quiz"
	"log"
	"time"
)

// TriggerDailyReviewsJob is a use case representing the entire daily job.
type TriggerDailyReviewsJob struct {
	userRepo         user.Repository
	wordRepo         word.Repository
	quizRepo         quiz.Repository
	createReviewQuiz quizCmd.CreateReviewQuizHandler
	notifier         notification.Notifier
}

// NewTriggerDailyReviewsJob creates a new job handler.
func NewTriggerDailyReviewsJob(
	userRepo user.Repository,
	wordRepo word.Repository,
	quizRepo quiz.Repository,
	createReviewQuiz quizCmd.CreateReviewQuizHandler,
	notifier notification.Notifier,
) *TriggerDailyReviewsJob {
	return &TriggerDailyReviewsJob{
		userRepo:         userRepo,
		wordRepo:         wordRepo,
		quizRepo:         quizRepo,
		createReviewQuiz: createReviewQuiz,
		notifier:         notifier,
	}
}

// Run executes the job logic.
func (j *TriggerDailyReviewsJob) Run() {
	log.Println("Starting daily review check job...")
	ctx := context.Background()

	userIDs, err := j.userRepo.FindAllIDs(ctx)
	if err != nil {
		log.Printf("ERROR: Could not fetch users for daily review job: %v", err)
		return
	}

	for _, userID := range userIDs {
		// In a real high-volume system, this would be parallelized with worker pools.
		j.processUserForReview(ctx, userID)
	}

	log.Println("Daily review check job finished.")
}

func (j *TriggerDailyReviewsJob) processUserForReview(ctx context.Context, userID uint) {
	domainUser, err := j.userRepo.FindByID(ctx, userID)
	if err != nil {
		log.Printf("ERROR: Could not get user %d details: %v", userID, err)
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
		log.Printf("ERROR: Could not get words due for review for user %d: %v", userID, err)
		return
	}

	if len(wordsDue) == 0 {
		return // No words to review, so do nothing.
	}

	// 3. If words are due, notify the user and show the main menu with the review button.
	message := fmt.Sprintf("👋 سلام! %d کلمه برای مرور روزانه شما آماده است.", len(wordsDue))
	if domainUser.LastMenu == "main" {
		if err := j.notifier.NotifyWithMainMenu(userID, message); err != nil {
			log.Printf("ERROR: Failed to send main menu notification to user %d: %v", userID, err)
		}
	} else {
		if err := j.notifier.Notify(userID, message); err != nil {
			log.Printf("ERROR: Failed to send simple review notification to user %d: %v", userID, err)
		}
	}
}
