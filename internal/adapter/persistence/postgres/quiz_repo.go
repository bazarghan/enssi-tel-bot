package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"gorm.io/gorm"
)

const (
	reviewQuizMaxOptions     = 4
	quizDefaultQuestionCount = 12
)

// QuizRepository is the GORM implementation of the quiz repository port.
type QuizRepository struct {
	db *gorm.DB
}

// NewQuizRepository creates a new repository.
func NewQuizRepository(db *gorm.DB) *QuizRepository {
	return &QuizRepository{db: db}
}

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
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(&newQuiz).Error; err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to create quiz db entry: %w", err)
	}

	questionsCreatedSuccessfully := 0
	for _, wordID := range wordIDs {
		if err := r.createQuestionInternal(tx, newQuiz.ID, wordID, courseID, wordIDs); err != nil {
			log.Printf("Error creating question for WordID %d (QuizID: %d): %v. Skipping.", wordID, newQuiz.ID, err)
		} else {
			questionsCreatedSuccessfully++
		}
	}

	if questionsCreatedSuccessfully == 0 {
		tx.Rollback()
		return 0, quiz.ErrQuizCreation
	}

	if newQuiz.QuestionCount != uint(questionsCreatedSuccessfully) {
		newQuiz.QuestionCount = uint(questionsCreatedSuccessfully)
		if err := tx.Model(&newQuiz).Update("question_count", newQuiz.QuestionCount).Error; err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to update quiz question count: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return 0, fmt.Errorf("failed to commit quiz creation transaction: %w", err)
	}

	return newQuiz.ID, nil
}

func (r *QuizRepository) FindActiveCourseBlockAttempt(ctx context.Context, userID, courseID uint, triggerProgress uint) (quiz.Attempt, error) {
	var am attemptModel
	err := r.db.WithContext(ctx).
		Joins("JOIN quizzes ON quizzes.id = quiz_attempts.quiz_id").
		Where("quiz_attempts.user_id = ? AND quizzes.course_id = ? AND quizzes.trigger_progress = ? AND quiz_attempts.is_completed = ? AND quizzes.type = ?",
			userID, courseID, triggerProgress, false, quiz.CourseBlock).
		Order("quiz_attempts.created_at DESC").
		First(&am).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return quiz.Attempt{}, quiz.ErrAttemptNotFound
		}
		return quiz.Attempt{}, err
	}

	return r.GetAttempt(ctx, am.ID)
}

func (r *QuizRepository) GetAttempt(ctx context.Context, attemptID uint) (quiz.Attempt, error) {
	var am attemptModel
	if err := r.db.WithContext(ctx).First(&am, attemptID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return quiz.Attempt{}, quiz.ErrAttemptNotFound
		}
		return quiz.Attempt{}, err
	}

	// Fetch the answers for this attempt
	var userAnswers []answerModel
	if err := r.db.WithContext(ctx).Where("quiz_attempt_id = ?", attemptID).Find(&userAnswers).Error; err != nil {
		// Non-fatal, we can still show the attempt without the answers
		log.Printf("Could not fetch user answers for attempt %d: %v", attemptID, err)
	}

	var qm quizModel
	err := r.db.WithContext(ctx).
		Preload("Questions.Options").
		First(&qm, am.QuizID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return quiz.Attempt{}, fmt.Errorf("quiz %d for attempt %d not found: %w", am.QuizID, attemptID, err)
		}
		return quiz.Attempt{}, err
	}

	return toDomainAttempt(am, qm, userAnswers), nil
}

func (r *QuizRepository) CreateAttempt(ctx context.Context, userID, quizID uint) (quiz.Attempt, error) {
	newAttempt := attemptModel{
		UserID: userID,
		QuizID: quizID,
	}
	if err := r.db.WithContext(ctx).Create(&newAttempt).Error; err != nil {
		return quiz.Attempt{}, err
	}
	return r.GetAttempt(ctx, newAttempt.ID)
}

func (r *QuizRepository) SaveAnswer(ctx context.Context, attemptID, questionID, optionID uint, isCorrect bool) error {
	tx := r.db.WithContext(ctx).Begin()
	var existingAnswerCount int64
	if err := tx.Model(&answerModel{}).
		Where("quiz_attempt_id = ? AND quiz_question_id = ?", attemptID, questionID).
		Count(&existingAnswerCount).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to check for existing answer: %w", err)
	}

	if existingAnswerCount > 0 {
		tx.Rollback()
		return quiz.ErrQuestionAlreadyAnswered
	}

	answer := answerModel{
		QuizAttemptID:        attemptID,
		QuizQuestionID:       questionID,
		QuizQuestionOptionID: optionID,
		IsCorrect:            isCorrect,
	}
	if err := tx.Create(&answer).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *QuizRepository) IncrementQuestionIndex(ctx context.Context, attemptID uint) error {
	return r.db.WithContext(ctx).Model(&attemptModel{}).Where("id = ?", attemptID).
		Update("current_question_num", gorm.Expr("current_question_num + 1")).Error
}

func (r *QuizRepository) MarkAttemptCompleted(ctx context.Context, attemptID uint) (int, error) {
	var score int
	tx := r.db.WithContext(ctx)

	// Calculate score
	if err := tx.Model(&answerModel{}).
		Where("quiz_attempt_id = ? AND is_correct = ?", attemptID, true).
		Select("COUNT(*)").
		Row().Scan(&score); err != nil {
		return 0, err
	}

	// Update attempt
	updates := map[string]interface{}{"is_completed": true, "score": score}
	if err := tx.Model(&attemptModel{}).Where("id = ?", attemptID).Updates(updates).Error; err != nil {
		return 0, err
	}

	return score, nil
}

func (r *QuizRepository) UpdateMessageID(ctx context.Context, attemptID uint, messageID int) error {
	result := r.db.WithContext(ctx).Model(&attemptModel{}).Where("id = ?", attemptID).Update("current_question_message_id", messageID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return quiz.ErrAttemptNotFound
	}
	return nil
}

func (r *QuizRepository) createQuestionInternal(tx *gorm.DB, quizID, wordID, courseID uint, distractorPoolIDs []uint) error {
	var questionWord wordModel
	if err := tx.First(&questionWord, wordID).Error; err != nil {
		return fmt.Errorf("failed to fetch word for question (WordID: %d): %w", wordID, err)
	}

	var questionWS wordSourceModel
	if err := tx.Where("word_id = ?", questionWord.ID).Preload("PartsOfSpeeches.Meanings").First(&questionWS).Error; err != nil {
		return fmt.Errorf("failed to fetch word source for question (WordID: %d): %w", questionWord.ID, err)
	}

	var allMeanings []string
	for _, pos := range questionWS.PartsOfSpeeches {
		for _, m := range pos.Meanings {
			if strings.TrimSpace(m.Title) != "" && (m.Lang == "Fa" || m.Lang == "En" || m.Lang == "") {
				allMeanings = append(allMeanings, m.Title)
			}
		}
	}
	if len(allMeanings) == 0 {
		return fmt.Errorf("word (ID: %d) has no usable meanings", questionWord.ID)
	}
	correctMeaningText := allMeanings[rand.Intn(len(allMeanings))]

	newQuestion := questionModel{
		QuizID: quizID,
		Text:   questionWord.Title,
		WordID: wordID,
	}
	if err := tx.Create(&newQuestion).Error; err != nil {
		return err
	}

	correctOption := optionModel{QuizQuestionID: newQuestion.ID, Text: correctMeaningText, IsCorrect: true}
	if err := tx.Create(&correctOption).Error; err != nil {
		return err
	}

	distractors, err := r.getDistractorMeanings(tx, correctMeaningText, wordID, courseID, distractorPoolIDs)
	if err != nil {
		log.Printf("Could not get distractors for word %d: %v", wordID, err)
	}

	for _, distractorText := range distractors {
		distractorOption := optionModel{QuizQuestionID: newQuestion.ID, Text: distractorText, IsCorrect: false}
		tx.Create(&distractorOption)
	}
	return nil
}

func (r *QuizRepository) getDistractorMeanings(tx *gorm.DB, correctText string, correctWordID, courseID uint, poolIDs []uint) ([]string, error) {
	localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	numDistractorsNeeded := reviewQuizMaxOptions - 1
	seenDistractors := make(map[string]bool)
	seenDistractors[correctText] = true
	finalDistractors := make([]string, 0, numDistractorsNeeded)

	// Step 1: Get from course block pool
	if courseID != 0 && len(poolIDs) > 0 {
		var meanings []string
		tx.Model(&meaningModel{}).
			Joins("JOIN part_of_speeches ON part_of_speeches.id = meanings.part_of_speech_id").
			Joins("JOIN word_sources ON word_sources.id = part_of_speeches.word_source_id").
			Joins("JOIN words ON words.id = word_sources.word_id").
			Joins("JOIN course_words ON course_words.word_id = words.id").
			Where("course_words.course_id = ? AND course_words.word_id IN ? AND course_words.word_id != ?", courseID, poolIDs, correctWordID).
			Pluck("meanings.title", &meanings)

		localRand.Shuffle(len(meanings), func(i, j int) { meanings[i], meanings[j] = meanings[j], meanings[i] })
		for _, m := range meanings {
			if len(finalDistractors) >= numDistractorsNeeded {
				break
			}
			if !seenDistractors[m] {
				finalDistractors = append(finalDistractors, m)
				seenDistractors[m] = true
			}
		}
	}

	// Step 2: Fill remaining from random words
	if len(finalDistractors) < numDistractorsNeeded {
		var randomMeanings []string
		tx.Model(&meaningModel{}).
			Where("title NOT IN ?", getKeys(seenDistractors)).
			Order(gorm.Expr("RANDOM()")).
			Limit(numDistractorsNeeded-len(finalDistractors)).
			Pluck("title", &randomMeanings)
		finalDistractors = append(finalDistractors, randomMeanings...)
	}

	return finalDistractors, nil
}

func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (r *QuizRepository) GetHighestScoreForCourseBlock(ctx context.Context, userID, courseID, triggerProgress uint) (int, error) {
	var maxScore sql.NullInt64 // Use sql.NullInt64 to handle cases where no rows are found (score is NULL)

	err := r.db.WithContext(ctx).Model(&attemptModel{}).
		Select("MAX(score)").
		Joins("JOIN quizzes ON quizzes.id = quiz_attempts.quiz_id").
		Where("quiz_attempts.user_id = ? AND quizzes.course_id = ? AND quizzes.trigger_progress = ? AND quiz_attempts.is_completed = ?",
			userID, courseID, triggerProgress, true).
		Row().Scan(&maxScore)

	if err != nil {
		// If no rows are found, gorm returns sql.ErrNoRows which we can treat as a score of 0.
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}

	if !maxScore.Valid {
		// This happens if MAX(score) returns NULL because no attempts were found.
		return 0, nil
	}

	return int(maxScore.Int64), nil
}
