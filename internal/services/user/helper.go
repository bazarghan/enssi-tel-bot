package user

import (
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gorm.io/gorm"
	"log"
	"time"
)

// ---------------------------- helper function createUserWithProfile ---------------------------------
func (s *Service) createUserWithProfile(telegramID int64, username, firstName, lastName string) (*models.User, error) {
	log.Printf("UserService: Creating new user for TelegramID %d.", telegramID)
	newUser := models.User{
		TelegramID: telegramID,
		LastMenu:   "main",
		LastActive: time.Now(),
		LastOnline: time.Now(),
		Profile: models.Profile{
			Username:  username,
			FirstName: firstName,
			LastName:  lastName,
		},
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if createErr := tx.Create(&newUser).Error; createErr != nil {
			return createErr
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w (TelegramID: %d): %w", ErrUserCreationFailed, telegramID, err)
	}

	log.Printf("UserService: Successfully created User ID: %d with Profile ID: %d", newUser.ID, newUser.Profile.ID)
	return &newUser, nil
}

// ---------------------------- helper function updateUserProfileIfNeeded ---------------------------------
func (s *Service) updateUserProfileIfNeeded(tx *gorm.DB, user *models.User, username, firstName, lastName string) error {

	profileChanged := false
	if user.Profile.ID == 0 && user.ID != 0 {
		var existingProfile models.Profile
		err := tx.Where("user_id = ?", user.ID).First(&existingProfile).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("UserService: Profile for existing User ID %d not found. Creating.", user.ID)
				user.Profile = models.Profile{
					UserID:    user.ID,
					Username:  username,
					FirstName: firstName,
					LastName:  lastName,
				}
			} else {
				return fmt.Errorf("%w for UserID %d: %w", ErrProfileFetchFailed, user.ID, err)
			}
		} else {
			user.Profile = existingProfile
		}
	}

	if user.Profile.Username != username {
		user.Profile.Username = username
		profileChanged = true
	}
	if user.Profile.FirstName != firstName {
		user.Profile.FirstName = firstName
		profileChanged = true
	}
	if user.Profile.LastName != lastName {
		user.Profile.LastName = lastName
		profileChanged = true
	}

	if profileChanged {
		log.Printf("UserService: Profile for UserID %d changed, will be updated during user save.", user.ID)
	}
	return nil
}

//--------------------------------------------------------------------------------------------------------------
