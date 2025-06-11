package jobs

import (
	"context"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/notification"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
	"github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/quiz"
	"log"
	"time"
)

// TriggerDailyReviewsJob is a use case representing the entire daily job.
type TriggerDailyReviewsJob struct {
	userRepo         user.Repository
	wordRepo         word.Repository
	createReviewQuiz quiz.CreateReviewQuizHandler
	notifier         notification.Notifier
}

// NewTriggerDailyReviewsJob creates a new job handler.
func NewTriggerDailyReviewsJob(
	userRepo user.Repository,
	wordRepo word.Repository,
	createReviewQuiz quiz.CreateReviewQuizHandler,
	notifier notification.Notifier,
) *TriggerDailyReviewsJob {
	return &TriggerDailyReviewsJob{
		userRepo:         userRepo,
		wordRepo:         wordRepo,
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
	// Check if the user already completed a review today.
	domainUser, err := j.userRepo.FindByID(ctx, userID)
	if err != nil {
		log.Printf("ERROR: Could not get user %d details: %v", userID, err)
		return
	}

	now := time.Now()
	if !domainUser.LastReviewSessionCompletedAt.IsZero() {
		lastReviewDate := domainUser.LastReviewSessionCompletedAt.In(now.Location())
		currentDate := now.In(now.Location())
		if lastReviewDate.Year() == currentDate.Year() && lastReviewDate.YearDay() == currentDate.YearDay() {
			return // Skip user, review already done today.
		}
	}

	// Get words due for review.
	wordsDue, err := j.wordRepo.GetWordsDueForReview(ctx, userID, now)
	if err != nil {
		log.Printf("ERROR: Could not get words due for review for user %d: %v", userID, err)
		return
	}

	if len(wordsDue) == 0 {
		return // No words to review.
	}

	// Create a review quiz.
	cmd := quiz.CreateReviewQuizCommand{UserID: userID, WordsToReview: wordsDue}
	_, err = j.createReviewQuiz.Handle(ctx, cmd)
	if err != nil {
		log.Printf("ERROR: Failed to create review quiz for user %d: %v", userID, err)
		return
	}

	// Notify the user.
	message := fmt.Sprintf("👋 سلام! %d کلمه برای مرور روزانه شما آماده است. برای شروع، وارد ربات شوید و به آزمون پاسخ دهید.", len(wordsDue))
	if err := j.notifier.Notify(userID, message); err != nil {
		log.Printf("ERROR: Failed to send review notification to user %d: %v", userID, err)
	}
}
