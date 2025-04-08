package models

import (
	"gorm.io/gorm"
)

type UserQuizWord struct {
	gorm.Model
	UserQuizID uint
	WordID     uint

	Score uint
}
