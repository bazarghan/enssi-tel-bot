package models

import (
	"gorm.io/gorm"
)

type Pronunciation struct {
	gorm.Model
	WordSourceID uint

	Region         string `gorm:"not null"`
	URL            string `gorm:"not null"`
	TelgramVoiceID string `gorm:"not null"`
}
