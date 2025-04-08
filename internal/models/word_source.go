package models

import (
	"gorm.io/gorm"
)

type WordSource struct {
	gorm.Model
	WordID          uint
	SourceID        uint
	Pronunciations  []Pronunciation
	Images          []Image
	PartsOfSpeeches []PartOfSpeech
	Phonetics       []Phonetic

	DefPrimary   string
	DefSecondary string
}
