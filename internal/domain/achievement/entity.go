package achievement

import (
	"github.com/bits-and-blooms/bitset"
	"time"
)

// Achievement defines a goal a user can accomplish.
type Achievement struct {
	ID              uint
	Title           string
	Description     string
	Type            string // e.g., "PROGRESSIVE_IMAGE"
	ImageURL        string // Path to base asset in assets/
	MinWordRequired uint
	TotalItems      uint
	GridWidth       uint
	GridHeight      uint
}

// UserAchievement tracks a user's progress towards a single achievement.
type UserAchievement struct {
	ID            uint
	UserID        uint
	AchievementID uint
	EarnedAt      time.Time
	State         *bitset.BitSet // Progress state for progressive achievements
}
