package models

import (
	"gorm.io/gorm"
)

type QuizAnswer struct {
	gorm.Model
	QuizAttemptID        uint
	QuizQuestionID       uint
	QuizQuestionOptionID uint
	IsCorrect            bool
}
