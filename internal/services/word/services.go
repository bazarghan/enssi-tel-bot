package word

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gorm.io/gorm"
)

// Service implements the WordService interface.
type Service struct {
	db *gorm.DB
}

// NewService creates a new instance of the word Service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Ensure Service implements WordService interface
var _ WordService = (*Service)(nil)

// GetWordDetailsForCourse retrieves and prepares a specific word from a course for display.
func (s *Service) GetWordDetailsForCourse(
	courseID uint,
	courseWordIndex uint,
	userID uint,
) (*WordDisplayData, error) {

	log.Printf("WordService: GetWordDetailsForCourse called for CourseID: %d, Index: %d, UserID: %d", courseID, courseWordIndex, userID)

	if courseID == 0 || courseWordIndex == 0 {
		return nil, fmt.Errorf("%w: courseID and courseWordIndex must be positive", ErrInvalidInput)
	}

	// 1. Fetch CourseWord
	var cw models.CourseWord
	err := s.db.Where("course_id = ? AND index = ?", courseID, courseWordIndex).First(&cw).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: no link for course %d at index %d", ErrCourseWordLinkNotFound, courseID, courseWordIndex)
		}
		return nil, fmt.Errorf("db error fetching course_word (course %d, index %d): %w", courseID, courseWordIndex, err)
	}

	// 2. Fetch the actual Word
	var word models.Word
	if err := s.db.First(&word, cw.WordID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: word_id %d linked from course_word not found", ErrWordNotFound, cw.WordID)
		}
		return nil, fmt.Errorf("db error fetching word %d: %w", cw.WordID, err)
	}

	// 3. Fetch its WordSource with preloaded details
	var ws models.WordSource
	err = s.db.
		Where("word_id = ?", word.ID).
		Preload("Phonetics").
		Preload("Pronunciations").
		Preload("PartsOfSpeeches.Meanings").
		First(&ws).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: for word_id %d", ErrWordSourceNotFound, word.ID)
		}
		return nil, fmt.Errorf("db error fetching word_source for word %d: %w", word.ID, err)
	}

	// 4. Format the word information
	formattedText, formatErr := s.formatWordForDisplay(&word, &ws)
	if formatErr != nil {
		log.Printf("WordService: Error formatting word ID %d: %v", word.ID, formatErr)
		return nil, fmt.Errorf("failed to prepare word display: %w", formatErr)
	}

	var imageURL, telegramImageID, telegramImageDocID string
	telegramImageID = cw.TelgramImageID
	telegramImageDocID = cw.TelgramImageDocID

	if telegramImageID == "" && len(ws.Images) > 0 {
		imageURL = ws.Images[0].URL
	}

	pronDisplayData := make([]PronunciationData, 0, len(ws.Pronunciations))
	for i := range ws.Pronunciations {
		p := &ws.Pronunciations[i]
		pronDisplayData = append(pronDisplayData, PronunciationData{
			ID:              p.ID,
			Region:          p.Region,
			TelegramVoiceID: p.TelgramVoiceID,
			AudioURL:        p.URL,
		})
	}

	displayData := &WordDisplayData{
		WordID:             word.ID,
		CourseID:           courseID,
		CourseWordIndex:    courseWordIndex,
		Title:              word.Title,
		FormattedText:      formattedText,
		TelegramImageID:    telegramImageID,
		TelegramImageDocID: telegramImageDocID,
		ImageURL:           imageURL,
		Pronunciations:     pronDisplayData,
	}

	return displayData, nil
}

// CacheTelegramFileIDForPronunciation updates a Pronunciation model with a Telegram File ID.
func (s *Service) CacheTelegramFileIDForPronunciation(pronunciationID uint, telegramFileID string) error {
	if pronunciationID == 0 || telegramFileID == "" {
		return fmt.Errorf("%w: pronunciationID and telegramFileID cannot be empty", ErrInvalidInput)
	}
	log.Printf("WordService: Caching TelegramFileID '%s' for PronunciationID %d", telegramFileID, pronunciationID)

	result := s.db.Model(&models.Pronunciation{}).Where("id = ?", pronunciationID).Update("telgram_voice_id", telegramFileID)
	if result.Error != nil {
		return fmt.Errorf("%w: updating pronunciation %d: %w", ErrFileCacheFailed, pronunciationID, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: pronunciation %d not found for caching file ID", ErrPronunciationNotFound, pronunciationID)
	}
	return nil
}

// CacheTelegramFileIDForCourseWordImage updates a CourseWord model with Telegram File IDs for its image.
func (s *Service) CacheTelegramFileIDForCourseWordImage(
	courseWordID uint, // Assuming courseWordID refers to the ID of the course_words table entry
	imageFileID string,
	imageDocFileID string,
) error {
	if courseWordID == 0 {
		return fmt.Errorf("%w: courseWordID cannot be empty", ErrInvalidInput)
	}
	if imageFileID == "" && imageDocFileID == "" {
		return fmt.Errorf("%w: at least one image file ID must be provided", ErrInvalidInput)
	}

	log.Printf("WordService: Caching ImageFileID '%s', ImageDocFileID '%s' for CourseWord.ID %d", imageFileID, imageDocFileID, courseWordID)

	updates := make(map[string]interface{})
	if imageFileID != "" {
		updates["telgram_image_id"] = imageFileID
	}
	if imageDocFileID != "" {
		updates["telgram_image_doc_id"] = imageDocFileID
	}

	log.Printf("WordService: CacheTelegramFileIDForCourseWordImage - SKIPPED due to CourseWord model structure (needs unique ID or query by composite key).")
	return nil // Placeholder
}

// MarkWordAsStudied records that a user has studied a specific word in a course context.
// This means the word enters or re-enters the Spaced Repetition System (SRS) with a 1-day recall.
func (s *Service) MarkWordAsStudied(userID uint, wordID uint, courseID uint) error {
	if userID == 0 || wordID == 0 { // courseID can be 0 if marking studied outside a course context, though less likely for this app
		return fmt.Errorf("%w: userID and wordID must be provided", ErrInvalidInput)
	}
	log.Printf("WordService: Marking word %d as studied for user %d (CourseContextID: %d)", wordID, userID, courseID)

	now := time.Now()
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		var wordStudied models.WordStudied
		err := tx.Where("user_id = ? AND word_id = ?", userID, wordID).First(&wordStudied).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// First time this user is studying this specific word in the SRS system
				wordStudied = models.WordStudied{
					UserID:             userID,
					WordID:             wordID,
					LastReviewdAt:      now,
					NextReviewAt:       now.Add(24 * time.Hour),
					ReviewIntervalDays: 1,
				}
				if createErr := tx.Create(&wordStudied).Error; createErr != nil {
					return fmt.Errorf("creating WordStudied record: %w", createErr)
				}
				log.Printf("WordService: Created new WordStudied for UserID %d, WordID %d. Next review in 1 day.", userID, wordID)
			} else {
				return fmt.Errorf("fetching WordStudied record: %w", err)
			}
		} else {
			// Word has been studied before. Reset its SRS schedule as it's being encountered in a course quiz.
			wordStudied.LastReviewdAt = now
			wordStudied.NextReviewAt = now.Add(24 * time.Hour)
			wordStudied.ReviewIntervalDays = 1
			if saveErr := tx.Save(&wordStudied).Error; saveErr != nil {
				return fmt.Errorf("updating WordStudied record: %w", saveErr)
			}
			log.Printf("WordService: Reset WordStudied SRS for UserID %d, WordID %d. Next review in 1 day.", userID, wordID)
		}

		// WordStudiedToday - for daily points/streaks, if courseID is relevant
		if courseID != 0 { // Only create WordStudiedToday if there's a course context
			todayWord := models.WordStudiedToday{
				UserID:   userID,
				WordID:   wordID,
				CourseID: courseID,
				Point:    1, // Example point
			}
			// Check if already studied today in this course to avoid duplicate points for the same interaction
			var existingToday models.WordStudiedToday
			todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			todayEnd := todayStart.Add(24 * time.Hour)

			errToday := tx.Where("user_id = ? AND word_id = ? AND course_id = ? AND created_at >= ? AND created_at < ?",
				userID, wordID, courseID, todayStart, todayEnd).
				First(&existingToday).Error

			if errors.Is(errToday, gorm.ErrRecordNotFound) {
				if createTodayErr := tx.Create(&todayWord).Error; createTodayErr != nil {
					// Log this error but don't fail the whole MarkWordAsStudied operation
					// as SRS update is more critical.
					log.Printf("WordService: Error creating WordStudiedToday record for UserID %d, WordID %d, CourseID %d: %v", userID, wordID, courseID, createTodayErr)
				}
			} else if errToday != nil {
				log.Printf("WordService: Error checking existing WordStudiedToday for UserID %d, WordID %d, CourseID %d: %v", userID, wordID, courseID, errToday)
			}
		}
		return nil
	})

	if txErr != nil {
		return fmt.Errorf("%w: %w", ErrMarkStudiedFailed, txErr)
	}
	return nil
}

// UpdateWordReviewSchedule updates the spaced repetition schedule for a word based on review performance.
func (s *Service) UpdateWordReviewSchedule(userID uint, wordID uint, wasCorrect bool) error {
	if userID == 0 || wordID == 0 {
		return fmt.Errorf("%w: userID and wordID must be provided", ErrInvalidInput)
	}
	log.Printf("WordService: Updating review schedule for UserID %d, WordID %d. Correct: %t", userID, wordID, wasCorrect)

	now := time.Now()
	return s.db.Transaction(func(tx *gorm.DB) error {
		var wordStudied models.WordStudied
		if err := tx.Where("user_id = ? AND word_id = ?", userID, wordID).First(&wordStudied).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("WordService: WordStudied record not found for UserID %d, WordID %d during UpdateWordReviewSchedule. This should not happen if review quizzes only contain studied words.", userID, wordID)
				return fmt.Errorf("%w: UserID %d, WordID %d", ErrWordStudiedNotFound, userID, wordID)
			}
			return fmt.Errorf("fetching WordStudied record for update: %w", err)
		}

		wordStudied.LastReviewdAt = now
		if wasCorrect {
			currentInterval := wordStudied.ReviewIntervalDays
			if currentInterval == 0 { // Should not happen if MarkWordAsStudied sets it to 1
				currentInterval = 1
			}
			nextInterval := currentInterval * 2
			// Define a maximum interval, e.g., 128 days
			if nextInterval > 128 {
				nextInterval = 128
			}
			wordStudied.ReviewIntervalDays = nextInterval
			wordStudied.NextReviewAt = now.Add(time.Duration(nextInterval) * 24 * time.Hour)
			log.Printf("WordService: Correct answer. WordID %d next review in %d days (At: %s)", wordID, nextInterval, wordStudied.NextReviewAt.Format(time.RFC3339))
		} else {
			wordStudied.ReviewIntervalDays = 1
			wordStudied.NextReviewAt = now.Add(24 * time.Hour)
			log.Printf("WordService: Incorrect answer. WordID %d next review in 1 day (At: %s)", wordID, wordStudied.NextReviewAt.Format(time.RFC3339))
		}

		if err := tx.Save(&wordStudied).Error; err != nil {
			return fmt.Errorf("saving updated WordStudied record: %w", err)
		}
		return nil
	})
}

// GetWordsDueForReview retrieves all WordStudied records for a user that are due for review by 'now'.
func (s *Service) GetWordsDueForReview(userID uint, now time.Time) ([]WordStudiedView, error) {
	if userID == 0 {
		return nil, fmt.Errorf("%w: userID must be provided", ErrInvalidInput)
	}
	log.Printf("WordService: Getting words due for review for UserID %d as of %s", userID, now.Format(time.RFC3339))

	var wordsStudied []models.WordStudied
	err := s.db.Where("user_id = ? AND next_review_at <= ?", userID, now).
		Order("next_review_at ASC"). // Optional: order by due time
		Find(&wordsStudied).Error

	if err != nil {
		return nil, fmt.Errorf("fetching due WordStudied records: %w", err)
	}

	if len(wordsStudied) == 0 {
		return []WordStudiedView{}, nil
	}

	wordIDs := make([]uint, len(wordsStudied))
	for i, ws := range wordsStudied {
		wordIDs[i] = ws.WordID
	}

	var words []models.Word
	if err := s.db.Where("id IN ?", wordIDs).Find(&words).Error; err != nil {
		return nil, fmt.Errorf("fetching word details for due words: %w", err)
	}

	wordMap := make(map[uint]models.Word)
	for _, w := range words {
		wordMap[w.ID] = w
	}

	resultViews := make([]WordStudiedView, 0, len(wordsStudied))
	for _, ws := range wordsStudied {
		wordDetail, ok := wordMap[ws.WordID]
		if !ok {
			log.Printf("WordService: Warning - Word details not found for WordID %d during GetWordsDueForReview, skipping.", ws.WordID)
			continue
		}
		resultViews = append(resultViews, WordStudiedView{
			UserID:             ws.UserID,
			WordID:             ws.WordID,
			WordTitle:          wordDetail.Title, // Assuming models.Word has a Title field
			NextReviewAt:       ws.NextReviewAt,
			ReviewIntervalDays: ws.ReviewIntervalDays,
		})
	}

	log.Printf("WordService: Found %d words due for review for UserID %d.", len(resultViews), userID)
	return resultViews, nil
}
