package models

import (
	"gorm.io/gorm"
)

type Meaning struct {
	gorm.Model
	PartOfSpeechID uint

	Title string `gorm:"not null"`
	Lang  string `gorm:"not null"`
}
