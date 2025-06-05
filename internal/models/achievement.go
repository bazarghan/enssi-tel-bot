package models

import (
	"gorm.io/gorm"
)

type Achievement struct {
	gorm.Model

	Profiles []ProfileAchievement `gorm:"foreignKey:AchievementID"`

	Title           string `gorm:"not null;unique"`
	Description     string
	Type            string
	ImageURL        string
	MinWordRequired uint
}
