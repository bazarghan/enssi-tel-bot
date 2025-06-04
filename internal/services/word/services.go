package word

import (
	"errors"
	"fmt"
	"log"
	"strings" // For pronunciation region check
	"time"    // For MarkWordAsStudied

	"github.com/2000ostd/enssi-tel-bot/internal/models"

	"gorm.io/gorm"
)

// Service implements the WordService interface.
type Service struct {
	db *gorm.DB
	// If you had an HTTP client for downloading audio, it would go here:
	// httpClient *http.Client
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
		Preload("Images"). // Assuming WordSource has an Images association
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
		// Log the specific formatting error, but return a more generic one or wrap it
		log.Printf("WordService: Error formatting word ID %d: %v", word.ID, formatErr)
		return nil, fmt.Errorf("failed to prepare word display: %w", formatErr)
	}

	// 5. Prepare image data
	var imageURL, telegramImageID, telegramImageDocID string
	telegramImageID = cw.TelgramImageID       // From CourseWord model
	telegramImageDocID = cw.TelgramImageDocID // From CourseWord model

	if telegramImageID == "" && len(ws.Images) > 0 {
		// Fallback to image from WordSource if CourseWord doesn't have a specific one cached
		// You might have specific logic to pick an image if multiple exist
		imageURL = ws.Images[0].URL
	}

	// 6. Prepare audio data (prioritize US, then UK, then first available with URL if no FileID)
	pronDisplayData := make([]PronunciationData, 0, len(ws.Pronunciations))
	var bestPronToUse *models.Pronunciation

	for i := range ws.Pronunciations {
		p := &ws.Pronunciations[i]
		pronDisplayData = append(pronDisplayData, PronunciationData{
			ID:              p.ID,
			Region:          p.Region,
			TelegramVoiceID: p.TelgramVoiceID,
			AudioURL:        p.URL,
		})

		// Logic to pick a primary pronunciation to suggest for download if none have TelegramVoiceID
		if p.TelgramVoiceID == "" && p.URL != "" { // Only consider those needing download
			if bestPronToUse == nil {
				bestPronToUse = p
			} else {
				// Prioritize US
				if strings.ToUpper(p.Region) == "US" && strings.ToUpper(bestPronToUse.Region) != "US" {
					bestPronToUse = p
				} else if strings.ToUpper(p.Region) == "UK" && strings.ToUpper(bestPronToUse.Region) != "US" && strings.ToUpper(bestPronToUse.Region) != "UK" {
					// Prioritize UK if current best is not US or UK
					bestPronToUse = p
				}
			}
		}
	}
	// The handler will decide whether to download if TelegramVoiceID is empty.
	// This service just provides the available data.

	// 7. Construct WordDisplayData
	displayData := &WordDisplayData{
		WordID:             word.ID,
		CourseID:           courseID,
		CourseWordIndex:    courseWordIndex,
		Title:              word.Title,
		FormattedText:      formattedText,
		TelegramImageID:    telegramImageID,
		TelegramImageDocID: telegramImageDocID,
		ImageURL:           imageURL, // Fallback if Telegram IDs are empty and an image URL exists
		Pronunciations:     pronDisplayData,
	}

	// Note: Marking word as studied is a separate action, typically called *after* successful display.
	// The calling service (e.g., CourseService) would call MarkWordAsStudied.

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
// This assumes your CourseWord model has a primary key ID. If it's a composite key, adjust accordingly.
func (s *Service) CacheTelegramFileIDForCourseWordImage(
	courseWordID uint,
	imageFileID string,
	imageDocFileID string,
) error {

	if courseWordID == 0 { // Or other primary key check
		return fmt.Errorf("%w: courseWordID cannot be empty", ErrInvalidInput)
	}
	if imageFileID == "" && imageDocFileID == "" {
		return fmt.Errorf("%w: at least one image file ID must be provided", ErrInvalidInput)
	}

	log.Printf("WordService: Caching ImageFileID '%s', ImageDocFileID '%s' for CourseWordID %d", imageFileID, imageDocFileID, courseWordID)

	updates := make(map[string]interface{})
	if imageFileID != "" {
		updates["telgram_image_id"] = imageFileID
	}
	if imageDocFileID != "" {
		updates["telgram_image_doc_id"] = imageDocFileID
	}

	// Assuming CourseWord has its own 'ID' primary key.
	// If CourseWord is identified by a composite key (CourseID, WordID) or (CourseID, Index),
	// the Where clause would need to change: s.db.Model(&models.CourseWord{}).Where("course_id = ? AND word_id = ?", cID, wID)
	// For this example, I'll assume CourseWord has a simple `ID`.
	// If not, you might need to pass CourseID and WordID/Index to identify the CourseWord row.
	// Let's assume you'd pass the direct primary key of the CourseWord join table entry if it has one.
	// If `CourseWord` uses a composite primary key like (course_id, word_id), the `modelType` parameter in the interface might be relevant
	// to distinguish, or better, have separate caching functions.
	// For this example, assuming `courseWordID` is the PK of the `course_words` table entry.

	result := s.db.Model(&models.CourseWord{}).Where("id = ?", courseWordID).Updates(updates) // Adjust 'id' if PK is different
	if result.Error != nil {
		return fmt.Errorf("%w: updating course_word %d image file IDs: %w", ErrFileCacheFailed, courseWordID, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: course_word %d not found for caching image file IDs", ErrCourseWordLinkNotFound, courseWordID)
	}
	return nil
}

// MarkWordAsStudied records that a user has studied a specific word in a course context.
func (s *Service) MarkWordAsStudied(userID uint, wordID uint, courseID uint) error {
	if userID == 0 || wordID == 0 || courseID == 0 {
		return fmt.Errorf("%w: userID, wordID, and courseID must be provided", ErrInvalidInput)
	}
	log.Printf("WordService: Marking word %d as studied for user %d in course %d", wordID, userID, courseID)

	// Using transaction to ensure both WordStudied and WordStudiedToday are updated atomically.
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. WordStudied (long-term tracking, spaced repetition)
		var wordStudied models.WordStudied
		err := tx.Where("user_id = ? AND word_id = ?", userID, wordID).First(&wordStudied).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// First time studying this word
				wordStudied = models.WordStudied{
					UserID:        userID,
					WordID:        wordID,
					LastReviewdAt: time.Now(),
					NextReviewAt:  time.Now().Add(24 * time.Hour), // Example: next review in 1 day
				}
				if createErr := tx.Create(&wordStudied).Error; createErr != nil {
					return fmt.Errorf("creating WordStudied record: %w", createErr)
				}
			} else {
				return fmt.Errorf("fetching WordStudied record: %w", err)
			}
		} else {
			// Word studied before, update review times
			wordStudied.LastReviewdAt = time.Now()
			// TODO: Implement actual spaced repetition logic to calculate NextReviewAt
			wordStudied.NextReviewAt = time.Now().Add(24 * 3 * time.Hour) // Example: next review in 3 days
			if saveErr := tx.Save(&wordStudied).Error; saveErr != nil {
				return fmt.Errorf("updating WordStudied record: %w", saveErr)
			}
		}

		// 2. WordStudiedToday (for daily streaks, points, etc.)
		// This might be simpler: just create a record for today.
		// If you need to aggregate points, that logic would be here.
		todayWord := models.WordStudiedToday{
			UserID:   userID,
			WordID:   wordID,
			CourseID: courseID,
			Point:    1, // Example: 1 point per word studied
		}
		// You might want to check if a record for this user/word/course already exists for *today*
		// to avoid duplicate points, or your table might have constraints.
		// For simplicity, creating a new record each time it's marked studied today.
		if createTodayErr := tx.Create(&todayWord).Error; createTodayErr != nil {
			return fmt.Errorf("creating WordStudiedToday record: %w", createTodayErr)
		}
		return nil
	})

	if txErr != nil {
		return fmt.Errorf("%w: %w", ErrMarkStudiedFailed, txErr)
	}
	return nil
}
