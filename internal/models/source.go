package models

import (
	"gorm.io/gorm"
)

type Source struct {
	gorm.Model
	Words []WordSource

	Title       string `gorm:"not null"`
	Description string
	URL         string `gorm:"not null"`
}
