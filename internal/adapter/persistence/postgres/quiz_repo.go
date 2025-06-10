package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"gorm.io/gorm"
	"log"
)

const reviewQuizMaxOptions = 4

// QuizRepository is the GORM implementation of the quiz repository port.
type QuizRepository struct {
	db *gorm.DB
}

// NewQuizRepository creates a new repository.
func NewQuizRepository(db *gorm.DB) *QuizRepository {
	return &QuizRepository{db: db}
}

// CreateQuiz creates the quiz structure.
func (r *QuizRepository) CreateQuiz(ctx context.Context, quizType quiz.QuizType, courseID, triggerProgress uint, wordIDs []uint) (uint, error) {
	if len(wordIDs) == 0 {
		return 0, quiz.ErrNoWordsForQuiz
	}

	newQuiz := quizModel{
		Type:            quizType,
		CourseID:        courseID,
		TriggerProgress: triggerProgress,
		QuestionCount:   uint(len(wordIDs)),
	}

	tx := r.db.WithContext(ctx).Begin()
	if err := tx.Create(&newQuiz).Error; err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to create quiz db entry: %w", err)
	}

	var distractorWordIDs []uint
	if newQuiz.Type == quiz.CourseBlock {
		distractorWordIDs = wordIDs
	}

	for _, wordID := range wordIDs {
		if err := r.createQuestionInternal(tx, newQuiz.ID, wordID, distractorWordIDs); err != nil {
			log.Printf("Error creating question for WordID %d (QuizID: %d): %v. Skipping.", wordID, newQuiz.ID, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return 0, fmt.Errorf("failed to commit quiz creation transaction: %w", err)
	}

	return newQuiz.ID, nil
}

// GetAttempt fetches a quiz attempt with its details.
func (r *QuizRepository) GetAttempt(ctx context.Context, attemptID uint) (quiz.Attempt, error) {
	var am attemptModel
	if err := r.db.WithContext(ctx).First(&am, attemptID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return quiz.Attempt{}, quiz.ErrAttemptNotFound
		}
		return quiz.Attempt{}, err
	}

	var qm quizModel
	err := r.db.WithContext(ctx).
		Preload("Questions.Options").
		First(&qm, am.QuizID).Error
	if err != nil {
		return quiz.Attempt{}, err
	}

	return toDomainAttempt(am, qm), nil
}

// CreateAttempt creates a new attempt record.
func (r *QuizRepository) CreateAttempt(ctx context.Context, userID, quizID uint) (quiz.Attempt, error) {
	newAttempt := attemptModel{
		UserID: userID,
		QuizID: quizID,
	}
	if err := r.db.WithContext(ctx).Create(&newAttempt).Error; err != nil {
		return quiz.Attempt{}, err
	}
	// Fetch the full attempt details to return.
	return r.GetAttempt(ctx, newAttempt.ID)
}

// ... other repository methods ...

// (This is a simplified internal helper, not part of the interface)
func (r *QuizRepository) createQuestionInternal(tx *gorm.DB, quizID, wordID uint, distractorPool []uint) error {
	var questionWord wordModel
	if err := tx.First(&questionWord, wordID).Error; err != nil {
		return fmt.Errorf("failed to fetch word for question: %w", err)
	}

	// Complex logic to get correct answer and distractors ported from old service...
	// For brevity, we'll create a placeholder structure.
	newQuestion := questionModel{
		QuizID: quizID,
		Text:   fmt.Sprintf("What is the meaning of '%s'?", questionWord.Title),
		WordID: wordID,
		Options: []optionModel{
			{Text: "Correct Answer", IsCorrect: true},
			{Text: "Distractor 1", IsCorrect: false},
			{Text: "Distractor 2", IsCorrect: false},
			{Text: "Distractor 3", IsCorrect: false},
		},
	}

	return tx.Create(&newQuestion).Error
}

// (The remaining methods of the repository would be implemented here by porting GORM logic from the old service)
