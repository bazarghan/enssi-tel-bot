package models

import (
	"gorm.io/gorm"
)

type Meaning struct {
	gorm.Model
	PartOfSpeechID uint

	Lang  string `gorm:"not null"`
	Title string `gorm:"not null"`
}
