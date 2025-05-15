package models

import (
	"gorm.io/gorm"
)

type Course struct {
	gorm.Model
	Users       []UserCourse
	CourseWords []CourseWord

	Title              string `gorm:"not null;check:length(title) >= 2"`
	PersianTitle       string `gorm:"not null;check:length(title) >= 2"`
	Description        string
	PersianDescription string
}
