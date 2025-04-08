package models

import (
	"gorm.io/gorm"
)

type WordStudiedToday struct {
	gorm.Model
	UserID   uint
	WordID   uint
	CourseID uint

	point int
}
