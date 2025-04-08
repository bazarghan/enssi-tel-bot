package models

import (
	"gorm.io/gorm"
)

type Word struct {
	gorm.Model
	Sources []WordSource

	Title string `gorm:"not null"`
	Lang  string `gorm:"not null"`
}
