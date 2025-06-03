package user

import (
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gorm.io/gorm"
	"log"
	"time"
)

//--------------------------------------------------------------------------------------------------------------

func (s *Service) GetOrCreateUserByTelegramID(telegramID int64, username string, firstName string, lastName string) (*models.User, error) {
	log.Printf("UserService: GetOrCreateUserByTelegramID called for TelegramID: %d, Username: %s", telegramID, username)
	var user models.User
	err := s.db.Preload("Profile").Where("telegram_id = ?", telegramID).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.createUserWithProfile(telegramID, username, firstName, lastName)
		}
		return nil, fmt.Errorf("error fetching user by telegram_id %d: %w", telegramID, err)
	}

	// --- User was found, update activity and profile if necessary ---
	log.Printf("UserService: User ID %d found for TelegramID %d.", user.ID, telegramID)
	user.LastActive = time.Now()
	user.LastOnline = time.Now()

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.updateUserProfileIfNeeded(tx, &user, username, firstName, lastName); err != nil {
			return fmt.Errorf("%w for user %d: %w", ErrUserFetchFailed, user.ID, err)
		}

		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("%w: for user_id %d: %w", ErrSaveUserFailed, user.ID, err)
		}
		return nil
	})

	if txErr != nil {
		return nil, txErr
	}
	return &user, nil
}
