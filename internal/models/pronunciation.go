package models

import (
	"gorm.io/gorm"
)

type Pronunciation struct {
	gorm.Model
	WordSourceID uint

	Lang string `gorm:"not null"`
	URL  string `gorm:"not null"`
}
