package models

import (
	"gorm.io/gorm"
)

type Phonetic struct {
	gorm.Model
	WordSourceID uint

	Lang  string `gorm:"not null"`
	Title string `gorm:"not null"`
}
