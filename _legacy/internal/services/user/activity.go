package user

import (
	"fmt"
	"log"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/models"
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
		// It's possible the user record was deleted between GetOrCreate and this call,
		// or an invalid userID was somehow passed.
		log.Printf("UserService: UpdateUserLastMenu - UserID %d not found or no changes made.", userID)
		return ErrUserNotFound // Be more specific if no rows affected
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
		"last_online": now, // Assuming last_online is also updated with general activity
	})
	if result.Error != nil {
		return fmt.Errorf("failed to record activity for user %d: %w", userID, result.Error)
	}
	if result.RowsAffected == 0 {
		log.Printf("UserService: RecordUserActivity - UserID %d not found or no changes made for activity timestamps.", userID)
		// Not returning ErrUserNotFound here as it might be called without strict existence check.
	}
	return nil
}

// MarkReviewSessionCompleted updates the user's timestamp for their last completed review session.
func (s *Service) MarkReviewSessionCompleted(userID uint, completedAt time.Time) error {
	log.Printf("UserService: MarkReviewSessionCompleted called for UserID: %d at %s", userID, completedAt.Format(time.RFC3339))
	if userID == 0 {
		return ErrInvalidUserID
	}
	if completedAt.IsZero() { // Ensure a valid time is provided
		log.Printf("UserService: MarkReviewSessionCompleted - received zero time for UserID %d, using current time.", userID)
		completedAt = time.Now()
	}

	result := s.db.Model(&models.User{}).Where("id = ?", userID).Update("last_review_session_completed_at", completedAt)
	if result.Error != nil {
		return fmt.Errorf("failed to update last_review_session_completed_at for user %d: %w", userID, result.Error)
	}
	if result.RowsAffected == 0 {
		log.Printf("UserService: MarkReviewSessionCompleted - UserID %d not found.", userID)
		return ErrUserNotFound
	}
	log.Printf("UserService: Successfully updated LastReviewSessionCompletedAt for UserID %d to %s", userID, completedAt.Format(time.RFC3339))
	return nil
}

