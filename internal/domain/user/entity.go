package user

import "time"

// Profile represents a user's personal information. It's a pure domain entity.
type Profile struct {
	ID        uint
	UserID    uint
	Username  string
	FirstName string
	LastName  string
	Score     uint
}

// User is the core domain entity for a user. It contains no framework tags.
type User struct {
	ID                           uint
	TelegramID                   int64
	IsAdmin                      bool
	LastMenu                     string
	LastReviewSessionCompletedAt time.Time
	Profile                      Profile
}
