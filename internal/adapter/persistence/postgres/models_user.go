package postgres

import (
	"gorm.io/gorm"
	"time"
)

// userModel is the GORM-specific struct for the 'users' table. It is unexported.
type UserModel struct {
	gorm.Model

	Profile ProfileModel `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	TelegramID                   int64 `gorm:"not null;uniqueIndex"`
	LastActive                   time.Time
	LastOnline                   time.Time
	LastMenu                     string    `gorm:"type:varchar(100);not null;default:''"`
	LastReviewSessionCompletedAt time.Time `gorm:"null"`
	IsAdmin                      bool      `gorm:"default:false"`
}

func (UserModel) TableName() string {
	return "users"
}

// profileModel is the GORM-specific struct for the 'profiles' table. It is unexported.
type ProfileModel struct {
	gorm.Model
	UserID    uint
	Username  string
	FirstName string
	LastName  string
	Score     uint
}

func (ProfileModel) TableName() string {
	return "profiles"
}
