package models

import (
	"time"

	"gorm.io/gorm"
)

type Profile struct {
	gorm.Model
	UserID       uint
	Achievements []UserAchievement

	Username    string `gorm:"type:string;check: length(username) >= 3 AND length(username) <= 32"`
	FirstName   string `gorm:"check:length(first_name) >= 2"`
	LastName    string `gorm:"check:length(last_name) >= 2"`
	DateOfBirth time.Time
	Score       uint `gorm:"check:score >= 0 AND score <= 100"`
}
