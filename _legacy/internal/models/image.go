package models

import (
	"gorm.io/gorm"
)

type Image struct {
	gorm.Model
	WordSourceID uint

	URL string `gorm:"not null"`
}
