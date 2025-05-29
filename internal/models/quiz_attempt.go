package models

import (
	"gorm.io/gorm"
)

type QuizAttempt struct {
	gorm.Model
	QuizID      uint `gorm:"not null"`
	UserID      uint `gorm:"not null"`
	Score       int  `gorm:"default:0"`
	IsCompleted bool `gorm:"default:false"`

	QuizAnswers []QuizAnswer

	CurrentQuestionNum       int `gorm:"default:0"`
	CurrentQuestionMessageID int
}
