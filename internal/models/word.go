package models

import (
	"gorm.io/gorm"
)

type Word struct {
	gorm.Model
	Sources []WordSource `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	Lang  string `gorm:"not null"`
	Title string `gorm:"not null"`
}
