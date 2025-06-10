package models

import (
	"gorm.io/gorm"
)

type WordSource struct {
	gorm.Model
	WordID   uint
	SourceID uint

	PartsOfSpeeches []PartOfSpeech  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Pronunciations  []Pronunciation `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Phonetics       []Phonetic      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Images          []Image         `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	DefPrimary   string
	DefSecondary string
}
