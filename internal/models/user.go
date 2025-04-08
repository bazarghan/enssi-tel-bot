package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model

	Profile           Profile `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Courses           []UserCourse
	UserQuizzes       []UserQuiz
	WordsStudiedToday []WordStudiedToday
	WordsStudied      []WordStudied

	Username   string `gorm:"type:string;uniqueIndex;not null;check: length(username) >= 5 AND length(username) <= 32"`
	LastActive time.Time
	LastOnline time.Time
}
