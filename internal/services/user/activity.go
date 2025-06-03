package user

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"log"
	"time"
)

// UpdateUserLastMenu updates the user's last known menu state.
func (s *Service) UpdateUserLastMenu(userID uint, menuState string) error {
	log.Printf("UserService: UpdateUserLastMenu called for UserID: %d, MenuState: %s", userID, menuState)
	if userID == 0 {
		return ErrInvalidUserID
	}
	result := s.db.Model(&models.User{}).Where("id = ?", userID).Update("last_menu", menuState)
	if result.Error != nil {
		return fmt.Errorf("failed to update last_menu for user %d: %w", userID, result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// RecordUserActivity updates timestamps like LastActive, LastOnline.
func (s *Service) RecordUserActivity(userID uint) error {
	log.Printf("UserService: RecordUserActivity called for UserID: %d", userID)
	if userID == 0 {
		return ErrInvalidUserID
	}
	now := time.Now()
	result := s.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"last_active": now,
		"last_online": now,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to record activity for user %d: %w", userID, result.Error)
	}
	if result.RowsAffected == 0 {
		log.Printf("UserService: RecordUserActivity - UserID %d not found or no changes made for activity timestamps.", userID)
	}
	return nil
}
