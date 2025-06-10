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
