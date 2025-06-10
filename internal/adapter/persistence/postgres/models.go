package postgres

import "gorm.io/gorm"

type userModel struct {
	gorm.Model
	Profile    profileModel `gorm:"foreignKey:UserID"`
	TelegramID int64        `gorm:"not null;uniqueIndex"`
	LastMenu   string
	IsAdmin    bool
}

type profileModel struct {
	gorm.Model
	UserID    uint `gorm:"uniqueIndex"`
	Username  string
	FirstName string
	LastName  string
	Score     uint
}
