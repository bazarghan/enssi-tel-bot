package models

import (
	"gorm.io/gorm"
)

type QuizQuestion struct {
	gorm.Model
	QuizID uint
	Text   string

	Options []QuizQuestionOption
}
