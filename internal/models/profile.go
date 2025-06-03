package models

import (
	"time"

	"gorm.io/gorm"
)

type Profile struct {
	gorm.Model
	UserID       uint
	Achievements []UserAchievement

	Username    string
	FirstName   string
	LastName    string
	DateOfBirth time.Time
	Score       uint
}
