package postgres

import (
	"context"
	"errors"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
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
	var cw courseWordModel
	err := r.db.WithContext(ctx).Where("course_id = ? AND index = ?", courseID, index).First(&cw).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return word.Word{}, word.ErrCourseWordLinkNotFound
		}
		return word.Word{}, err
	}

	var wm wordModel
	if err := r.db.WithContext(ctx).First(&wm, cw.WordID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return word.Word{}, word.ErrNotFound
		}
		return word.Word{}, err
	}

	var wsm wordSourceModel
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
	var model studiedWordModel
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
