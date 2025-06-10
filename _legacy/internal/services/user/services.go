package user

import (
	"gorm.io/gorm"
)

// Service implements the UserService interface.
type Service struct {
	db *gorm.DB
}

// NewService creates a new instance of the user Service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}
