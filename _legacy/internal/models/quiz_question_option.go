package models

import (
	"gorm.io/gorm"
)

type QuizQuestionOption struct {
	gorm.Model
	QuizQuestionID uint   `gorm:"not null"`
	Text           string `gorm:"not null"`
	IsCorrect      bool   `gorm:"not null;default:false"`
}
