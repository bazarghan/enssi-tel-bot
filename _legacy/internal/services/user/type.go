// internal/services/user/types.go
package user

import (
	"errors"
	"time"

	"github.com/bits-and-blooms/bitset"
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
	ErrInvalidInput       = errors.New("user service: the input is invalid")
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
	ID          uint // The ID of the achievement itself
	Title       string
	Description string
	Type        string // e.g., "PROGRESSIVE_IMAGE_504"
	ImageURL    string // Path to the base image for generation or static display
	EarnedOn    time.Time

	// Fields required for progressive achievements
	TotalItems uint
	GridWidth  uint
	GridHeight uint

	// User-specific progress for this achievement
	StateBitSet *bitset.BitSet
}

type UpdateProfileRequest struct {
	FirstName   *string
	LastName    *string
	DateOfBirth *time.Time
}
