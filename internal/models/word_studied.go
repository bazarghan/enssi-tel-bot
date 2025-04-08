package models

import (
	"time"

	"gorm.io/gorm"
)

type WordStudied struct {
	gorm.Model
	UserID        uint
	WordID        uint
	LastReviewdAt time.Time
	NextReviewAt  time.Time
}
