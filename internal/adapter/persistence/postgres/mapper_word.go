package postgres

import "github.com/2000ostd/enssi-tel-bot/internal/domain/word"

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
