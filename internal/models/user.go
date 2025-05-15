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

	TelegramID int64 `gorm:"not null;uniqueIndex"`
	LastActive time.Time
	LastOnline time.Time
	LastMenu   string `gorm:"type:varchar(100);not null;default:''"`
}
