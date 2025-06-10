// internal/domain/user/entity.go
package user

import "time"

// User is the core business entity for a user.
type User struct {
	ID         uint
	Profile    Profile
	TelegramID int64
	LastMenu   string
	IsAdmin    bool
	CreatedAt  time.Time
}

// Profile holds user-specific, non-auth information.
type Profile struct {
	ID        uint
	UserID    uint
	Username  string
	FirstName string
	LastName  string
	Score     uint
}
