package models

import (
	"gorm.io/gorm"
)

type QuizAnswer struct {
	gorm.Model
	QuizAttemptID uint
	QuestionID    uint
	OptionID      uint
	IsCorrect     bool
}
