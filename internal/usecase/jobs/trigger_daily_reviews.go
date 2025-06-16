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
	// --- NEW: Check for and DELETE any old pending review quiz ---
	pendingReview, err := j.quizRepo.FindPendingReviewAttempt(ctx, userID)
	if err == nil && pendingReview.ID != 0 {
		log.Printf("User %d has an old, incomplete daily review (AttemptID: %d). Deleting it to create a fresh one.", userID, pendingReview.ID)
		if deleteErr := j.quizRepo.DeleteAttempt(ctx, pendingReview.ID); deleteErr != nil {
			log.Printf("ERROR: Failed to delete old pending review attempt %d: %v", pendingReview.ID, deleteErr)
			// We still continue, as we might be able to create a new one anyway.
		}
	}
	// --- END OF NEW LOGIC ---

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
	cmd := quizCmd.CreateReviewQuizCommand{UserID: userID, WordsToReview: wordsDue}
	_, err = j.createReviewQuiz.Handle(ctx, cmd)
	if err != nil {
		log.Printf("ERROR: Failed to create review quiz for user %d: %v", userID, err)
		return
	}

	// Notify the user.
	message := fmt.Sprintf("سلام %d کلمه برای مرور روزانه شما آماده است برای مرور در منوی اصلی رو دکمه مرور روزانه بزنید.", len(wordsDue))
	// --- NEW LOGIC: Always send the main menu and reset the user's state ---

	// 1. Send the notification with the full, updated main menu.
	log.Printf("Sending updated main menu with review button to user %d", userID)
	if err := j.notifier.NotifyWithMainMenu(userID, message); err != nil {
		log.Printf("ERROR: Failed to send main menu notification to user %d: %v", userID, err)
		return // Stop if we can't notify the user
	}

	// 2. Update the user's state in the database to 'main'.
	if err := j.userRepo.UpdateLastMenu(ctx, userID, "main"); err != nil {
		log.Printf("ERROR: Failed to update user menu state for user %d after sending review notification: %v", userID, err)
	}
	// --- END OF NEW LOGIC ---

}
