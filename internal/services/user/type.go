// internal/services/user/types.go
package user

import (
	"errors"
	"time"
)

// --- Service-Specific Errors ---
var (
	ErrNotImplemented  = errors.New("user service: function not implemented")
	ErrUserNotFound    = errors.New("user service: user not found")
	ErrProfileNotFound = errors.New("user service: user profile not found")

	ErrInvalidProfileData = errors.New("user service: invalid data for profile update")
	ErrInvalidUserID      = errors.New("user service: invalid user ID (cannot be zero)")

	ErrSaveUserFailed     = errors.New("user service: failed to save user data")
	ErrUserCreationFailed = errors.New("user service: failed to create new user and profile")
	ErrProfileFetchFailed = errors.New("user service: failed to fetch profile data")
	ErrUserFetchFailed    = errors.New("user service: failed to fetch user data")
)

// --- Request/Response Structs for Service Methods ---
type UserProfileView struct {
	UserID        uint
	Username      string
	FirstName     string
	LastName      string
	DateOfBirth   *time.Time
	Score         uint
	WordsStudied  int
	CoursesActive int
	Achievements  []AchievementView
}

type AchievementView struct {
	Title       string
	Description string
	ImageURL    string
	EarnedOn    time.Time
}

type UpdateProfileRequest struct {
	FirstName   *string
	LastName    *string
	DateOfBirth *time.Time
}
