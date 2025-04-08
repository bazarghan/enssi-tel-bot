package models

import (
	"gorm.io/gorm"
)

type Phonetic struct {
	gorm.Model
	WordSourceID uint

	Title string `gorm:"not null"`
	Lang  string `gorm:"not null"`
}
