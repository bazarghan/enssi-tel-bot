package models

type CourseWord struct {
	CourseID          uint
	WordID            uint
	TelgramImageID    string `gorm:"not null`
	TelgramImageDocID string `gorm:"not null`
	Lesson            string
	Index             uint
}
