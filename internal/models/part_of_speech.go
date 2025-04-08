package models

import (
	"gorm.io/gorm"
)

type PartOfSpeech struct {
	gorm.Model
	WordSourceID uint

	Title    string `gorm:"not null"`
	Meanings []Meaning
}
