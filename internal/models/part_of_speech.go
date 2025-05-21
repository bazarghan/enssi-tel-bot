package models

import (
	"gorm.io/gorm"
)

type PartOfSpeech struct {
	gorm.Model
	WordSourceID uint

	Meanings []Meaning `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	Title string `gorm:"not null"`
}
