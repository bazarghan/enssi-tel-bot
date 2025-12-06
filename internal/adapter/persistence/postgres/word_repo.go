package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/bazarghan/enssi-tel-bot/internal/domain/word"
	"gorm.io/gorm"
)

// WordRepository is the GORM implementation of the word repository port.
type WordRepository struct {
	db *gorm.DB
}

// NewWordRepository creates a new repository.
func NewWordRepository(db *gorm.DB) *WordRepository {
	return &WordRepository{db: db}
}

// FindByCourseIndex finds a single word for a course at a specific index.
func (r *WordRepository) FindByCourseIndex(ctx context.Context, courseID uint, index uint) (word.Word, error) {
	var cw CourseWordModel
	err := r.db.WithContext(ctx).Where("course_id = ? AND index = ?", courseID, index).First(&cw).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return word.Word{}, word.ErrCourseWordLinkNotFound
		}
		return word.Word{}, err
	}

	var wm WordModel
	if err := r.db.WithContext(ctx).First(&wm, cw.WordID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return word.Word{}, word.ErrNotFound
		}
		return word.Word{}, err
	}

	var wsm WordSourceModel
	err = r.db.WithContext(ctx).
		Where("word_id = ?", wm.ID).
		Preload("Phonetics").
		Preload("Pronunciations").
		Preload("PartsOfSpeeches.Meanings").
		Preload("Images").
		First(&wsm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return word.Word{}, word.ErrSourceNotFound
		}
		return word.Word{}, err
	}

	return toDomainWord(wm, wsm), nil
}

// FindStudiedWord retrieves a user's SRS data for a specific word.
func (r *WordRepository) FindStudiedWord(ctx context.Context, userID, wordID uint) (word.StudiedWord, error) {
	var model StudiedWordModel
	err := r.db.WithContext(ctx).Where("user_id = ? AND word_id = ?", userID, wordID).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return a zero-value object and a special error to indicate "not found"
			return word.StudiedWord{}, word.ErrStudiedWordNotFound
		}
		return word.StudiedWord{}, err
	}
	return toDomainStudiedWord(model), nil
}

// SaveStudiedWord creates or updates a user's SRS data for a word.
func (r *WordRepository) SaveStudiedWord(ctx context.Context, sw word.StudiedWord) error {
	model := toPersistenceStudiedWord(sw)
	// gorm's Save handles both create (if ID is 0) and update (if ID is non-zero).
	return r.db.WithContext(ctx).Save(&model).Error
}

// FindWordIDsByCourseBlock retrieves a slice of word IDs for a specific block in a course.
func (r *WordRepository) FindWordIDsByCourseBlock(ctx context.Context, courseID uint, limit uint, offset uint) ([]uint, error) {
	var wordIDs []uint
	err := r.db.WithContext(ctx).Model(&CourseWordModel{}).
		Where("course_id = ?", courseID).
		Order("index ASC").
		Limit(int(limit)).
		Offset(int(offset)).
		Pluck("word_id", &wordIDs).Error
	return wordIDs, err
}

// GetWordsDueForReview retrieves all WordStudied records for a user that are due for review by 'now'.
func (r *WordRepository) GetWordsDueForReview(ctx context.Context, userID uint, now time.Time) ([]word.StudiedWord, error) {
	var models []StudiedWordModel
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND next_review_at <= ?", userID, now).
		Find(&models).Error

	if err != nil {
		return nil, err
	}

	domainStudiedWords := make([]word.StudiedWord, len(models))
	for i, m := range models {
		domainStudiedWords[i] = toDomainStudiedWord(m)
	}

	return domainStudiedWords, nil
}

func (r *WordRepository) FindDisplayableWordByIndex(ctx context.Context, courseID uint, index uint) (word.DisplayableWord, error) {
	var cw CourseWordModel
	err := r.db.WithContext(ctx).Where("course_id = ? AND index = ?", courseID, index).First(&cw).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return word.DisplayableWord{}, word.ErrCourseWordLinkNotFound
		}
		return word.DisplayableWord{}, err
	}

	domainWord, err := r.FindByCourseIndex(ctx, courseID, index)
	if err != nil {
		return word.DisplayableWord{}, err
	}

	displayable := word.DisplayableWord{
		Word:               domainWord,
		CourseWordID:       cw.ID, // Assuming courseWordModel has gorm.Model
		TelegramImageID:    cw.TelgramImageID,
		TelegramImageDocID: cw.TelgramImageDocID,
	}

	for _, pron := range domainWord.Pronunciations {
		// This is slightly inefficient but works. A single complex query would be better in a high-load system.
		var pronModel PronunciationModel
		r.db.WithContext(ctx).First(&pronModel, pron.ID)
		displayable.Pronunciations = append(displayable.Pronunciations, word.DisplayablePronunciation{
			Pronunciation:   pron,
			TelegramVoiceID: pronModel.TelgramVoiceID,
		})
	}

	return displayable, nil
}

func (r *WordRepository) CacheImageFileIDs(ctx context.Context, courseWordID uint, imageFileID, imageDocFileID string) error {
	updates := make(map[string]interface{})
	if imageFileID != "" {
		updates["telgram_image_id"] = imageFileID
	}
	if imageDocFileID != "" {
		updates["telgram_image_doc_id"] = imageDocFileID
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&CourseWordModel{}).Where("id = ?", courseWordID).Updates(updates).Error
}

func (r *WordRepository) CacheVoiceFileID(ctx context.Context, pronunciationID uint, voiceFileID string) error {
	return r.db.WithContext(ctx).Model(&PronunciationModel{}).Where("id = ?", pronunciationID).Update("telgram_voice_id", voiceFileID).Error
}

func (r *WordRepository) CountMasteredWords(ctx context.Context, userID uint) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&StudiedWordModel{}).
		Where("user_id = ? AND is_mastered = ?", userID, true).
		Count(&count).Error
	return int(count), err
}

// CountTotalStudiedWords counts the total number of studied word records across all users.
func (r *WordRepository) CountTotalStudiedWords(ctx context.Context) (int64, error) {
	var count int64
	// This counts every entry in the word_studieds table, representing each time any user has studied any word.
	err := r.db.WithContext(ctx).Model(&StudiedWordModel{}).Count(&count).Error
	return count, err
}

// CountStudiedWords counts the number of words studied by a specific user.
func (r *WordRepository) CountStudiedWords(ctx context.Context, userID uint) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&StudiedWordModel{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return int(count), err
}
