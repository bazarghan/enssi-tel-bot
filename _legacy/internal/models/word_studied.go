package models

import (
	"time"

	"gorm.io/gorm"
)

type WordStudied struct {
	gorm.Model
	UserID uint `gorm:"index"`
	WordID uint `gorm:"index"`

	LastReviewdAt      time.Time
	NextReviewAt       time.Time `gorm:"index"`
	ReviewIntervalDays uint      `gorm:"default:1"`
}
