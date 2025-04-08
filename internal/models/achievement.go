package models

import (
	"gorm.io/gorm"
)

type Achievement struct {
	gorm.Model
	Profiles []UserAchievement

	Title           string `gorm:"not null;unique"`
	Description     string
	Type            string `gorm:"not null"`
	ImageURL        string
	MinWordRequired uint
}
