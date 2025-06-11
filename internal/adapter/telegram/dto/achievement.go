package dto

import "time"

// AchievementView is a DTO for listing achievements to the user.
type AchievementView struct {
	ID          uint
	Title       string
	Description string
	EarnedOn    time.Time
}
