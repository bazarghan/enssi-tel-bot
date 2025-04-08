package models

import (
	"gorm.io/gorm"
)

type UserQuiz struct {
	gorm.Model
	UserID        uint
	CourseID      uint
	UserQuizWords []UserQuizWord

	Type        string
	IsCompleted bool `gorm:"default:false"`
	Score       uint `gorm:"check:score >= 0 AND score <= 100"`
}
