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

	TotalItems uint `gorm:"comment:Total items/pixels/blocks to unlock for this achievement"`
	GridWidth  uint `gorm:"comment:For image grid achievements, the width of the grid in items/blocks"`
	GridHeight uint `gorm:"comment:For image grid achievements, the height of the grid in items/blocks"`
}
