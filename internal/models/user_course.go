package models

import (
	"gorm.io/gorm"
)

type UserCourse struct {
	gorm.Model
	UserID   uint
	CourseID uint
	Progress uint `gorm:"check:progress >= 0 AND progress <= 100"`
}
