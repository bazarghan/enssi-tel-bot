package models

import (
	"gorm.io/gorm"
)

type QuizQuestion struct {
	gorm.Model
	QuizID  uint
	Text    string
	Options []QuizQuestionOption
	WordID  uint `gorm:"index;comment:The ID of the word this question is about, especially for review quizzes"`
}
