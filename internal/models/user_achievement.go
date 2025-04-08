package models

import (
	"gorm.io/gorm"
)

type UserAchievement struct {
	gorm.Model
	ProfileID     uint
	AchievementID uint
}
