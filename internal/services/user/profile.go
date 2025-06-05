package user

import (
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gorm.io/gorm"
	"log"
)

// UpdateUserProfile allows updating mutable parts of a user's profile.
func (s *Service) UpdateUserProfile(userID uint, req UpdateProfileRequest) error {
	log.Printf("UserService: UpdateUserProfile called for UserID: %d, Request: %+v", userID, req)
	if userID == 0 {
		return ErrInvalidUserID
	}

	var profile models.Profile
	err := s.db.Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProfileNotFound
		}
		return fmt.Errorf("%w for update (UserID %d): %w", ErrProfileFetchFailed, userID, err)
	}

	updates := make(map[string]interface{})
	madeChanges := false
	if req.FirstName != nil && profile.FirstName != *req.FirstName {
		updates["first_name"] = *req.FirstName
		madeChanges = true
	}
	if req.LastName != nil && profile.LastName != *req.LastName {
		updates["last_name"] = *req.LastName
		madeChanges = true
	}
	if req.DateOfBirth != nil && (profile.DateOfBirth.IsZero() || !profile.DateOfBirth.Equal(*req.DateOfBirth)) {
		updates["date_of_birth"] = *req.DateOfBirth
		madeChanges = true
	}

	if !madeChanges {
		log.Printf("UserService: No effective changes to update for profile of UserID: %d", userID)
		return nil
	}

	log.Printf("UserService: Updating profile for UserID %d with: %+v", userID, updates)
	// Use transaction for updating profile to ensure atomicity if needed,
	// though for a single Updates call, it's less critical than multi-step operations.
	err = s.db.Model(&profile).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("%w: for profile of UserID %d: %w", ErrSaveUserFailed, userID, err)
	}

	return nil
}

// GetUserProfile retrieves the profile information for a user.
func (s *Service) GetUserProfile(userID uint) (*UserProfileView, error) {

	log.Printf("UserService: GetUserProfile called for UserID: %d", userID)
	if userID == 0 {
		return nil, ErrInvalidUserID
	}

	var user models.User
	err := s.db.
		Preload("Profile").
		Preload("Profile.Achievements.Achievement"). // Corrected Preload path
		First(&user, userID).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("%w for profile view (ID %d): %w", ErrUserFetchFailed, userID, err)
	}

	if user.Profile.ID == 0 {
		// This implies User exists but has no Profile, which should ideally not happen
		// if GetOrCreateUserByTelegramID always ensures a Profile is made.
		log.Printf("UserService: Warning - UserID %d exists but has no associated Profile record.", userID)
		return nil, ErrProfileNotFound
	}

	// ---> START OF CALCULATIONS <---

	// Calculate WordsStudied
	var wordsStudiedCount int64 // GORM's Count returns int64
	if err := s.db.Model(&models.WordStudied{}).Where("user_id = ?", userID).Count(&wordsStudiedCount).Error; err != nil {
		log.Printf("UserService: Error counting words studied for UserID %d: %v", userID, err)
		// Decide how to handle this error: return error, or proceed with 0, or log and proceed.
		// For now, let's log and proceed, wordsStudiedCount will remain 0 if there's an error.
	}

	// Calculate CoursesActive
	var userCourses []models.UserCourse
	if err := s.db.Where("user_id = ?", userID).Find(&userCourses).Error; err != nil {
		log.Printf("UserService: Error fetching user courses for UserID %d: %v", userID, err)
	}
	activeCoursesCount := len(userCourses)
	// ---> END OF CALCULATIONS <---

	profileView := UserProfileView{
		UserID:        user.ID,
		Username:      user.Profile.Username,
		FirstName:     user.Profile.FirstName,
		LastName:      user.Profile.LastName,
		DateOfBirth:   nil,
		Score:         user.Profile.Score,
		WordsStudied:  int(wordsStudiedCount),
		CoursesActive: activeCoursesCount,
		// this is just place holder later ToDo fix this part
		Achievements: make([]AchievementView, 0, len(user.Profile.Achievements)),
	}
	if !user.Profile.DateOfBirth.IsZero() {
		dob := user.Profile.DateOfBirth
		profileView.DateOfBirth = &dob
	}

	return &profileView, nil
}
