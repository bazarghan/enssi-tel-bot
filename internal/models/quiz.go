package models

import (
	"gorm.io/gorm"
)

type QuizType string

const (
	QuizTypeCourseBlock QuizType = "COURSE_BLOCK" // Standard quiz after a block of course words
	QuizTypeReview      QuizType = "REVIEW"       // Spaced repetition review quiz
)

type Quiz struct {
	gorm.Model
	CourseID       uint
	QuizQuesetions []QuizQuestion

	QuestionCount   uint
	Type            QuizType `gorm:"type:varchar(50);default:'COURSE_BLOCK'"`
	TriggerProgress uint
}
