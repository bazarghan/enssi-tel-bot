package models

import (
	"github.com/bits-and-blooms/bitset"
	"gorm.io/gorm"
)

type UserAchievement struct {
	gorm.Model
	ProfileID     uint
	AchievementID uint
	State         bitset.BitSet
}
