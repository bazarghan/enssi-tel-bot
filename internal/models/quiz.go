package models

import (
	"gorm.io/gorm"
)

type Quiz struct {
	gorm.Model
	CourseID       uint
	QuizQuesetions []QuizQuestion

	Type string
}
