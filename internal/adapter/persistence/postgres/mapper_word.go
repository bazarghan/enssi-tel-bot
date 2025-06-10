package postgres

import (
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
	"gorm.io/gorm"
	"time"
)

// toDomainWord converts the GORM wordSourceModel to the pure domain Word entity.
func toDomainWord(wm wordModel, wsm wordSourceModel) word.Word {
	domainWord := word.Word{
		ID:           wm.ID,
		Title:        wm.Title,
		PrimaryDef:   wsm.DefPrimary,
		SecondaryDef: wsm.DefSecondary,
	}

	if len(wsm.Phonetics) > 0 {
		domainWord.Phonetic = wsm.Phonetics[0].Title
	}
	if len(wsm.Images) > 0 {
		domainWord.ImageURL = wsm.Images[0].URL
	}

	for _, psm := range wsm.PartsOfSpeeches {
		pos := word.PartOfSpeech{Title: psm.Title}
		for _, mm := range psm.Meanings {
			pos.Meanings = append(pos.Meanings, word.Meaning{
				Lang:  mm.Lang,
				Title: mm.Title,
			})
		}
		domainWord.PartsOfSpeech = append(domainWord.PartsOfSpeech, pos)
	}

	for _, prsm := range wsm.Pronunciations {
		domainWord.Pronunciations = append(domainWord.Pronunciations, word.Pronunciation{
			ID:       prsm.ID,
			Region:   prsm.Region,
			AudioURL: prsm.URL,
		})
	}

	return domainWord
}

func toDomainStudiedWord(m studiedWordModel) word.StudiedWord {
	return word.StudiedWord{
		ID:                 m.ID,
		UserID:             m.UserID,
		WordID:             m.WordID,
		LastReviewedAt:     m.LastReviewedAt,
		NextReviewAt:       m.NextReviewAt,
		ReviewIntervalDays: m.ReviewIntervalDays,
	}
}

func toPersistenceStudiedWord(d word.StudiedWord) studiedWordModel {
	// If ID is 0, GORM will INSERT. If ID is non-zero, it will UPDATE.
	// We explicitly clear time fields for GORM to handle them correctly on create vs update.
	var createdAt, updatedAt time.Time
	if d.ID != 0 {
		// Keep original created at time if we are updating.
		// This requires fetching first, which our logic does.
	}

	return studiedWordModel{
		Model:              gorm.Model{ID: d.ID, CreatedAt: createdAt, UpdatedAt: updatedAt},
		UserID:             d.UserID,
		WordID:             d.WordID,
		LastReviewedAt:     d.LastReviewedAt,
		NextReviewAt:       d.NextReviewAt,
		ReviewIntervalDays: d.ReviewIntervalDays,
	}
}
