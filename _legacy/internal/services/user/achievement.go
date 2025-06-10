package user

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"github.com/bits-and-blooms/bitset"

	"gorm.io/gorm"
)

// AwardAchievementProgress updates the state of a progressive achievement for a user.
// It reveals a specified number of items (e.g., pixels or blocks) by finding the
// next available unset bits and setting them.
func (s *Service) AwardAchievementProgress(userID uint, achievementID uint, itemsToReveal int) (*models.ProfileAchievement, *bitset.BitSet, error) {
	if userID == 0 || achievementID == 0 || itemsToReveal <= 0 {
		return nil, nil, fmt.Errorf("%w: userID, achievementID must be positive, and itemsToReveal must be greater than 0", ErrInvalidInput)
	}

	log.Printf("UserService: Awarding achievement progress for UserID %d, AchievementID %d, Items to reveal: %d", userID, achievementID, itemsToReveal)

	var profile models.Profile
	if err := s.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("profile not found for UserID %d: %w", userID, ErrProfileNotFound)
		}
		return nil, nil, fmt.Errorf("failed to fetch profile for UserID %d: %w", userID, err)
	}

	var achievementDetails models.Achievement
	if err := s.db.First(&achievementDetails, achievementID).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to fetch achievement details for AchievementID %d: %w", achievementID, err)
	}

	var pa models.ProfileAchievement
	err := s.db.Where("profile_id = ? AND achievement_id = ?", profile.ID, achievementID).
		First(&pa).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("UserService: No existing ProfileAchievement for ProfileID %d, AchievementID %d. Creating new one.", profile.ID, achievementID)

			pa = models.ProfileAchievement{
				ProfileID:     profile.ID,
				AchievementID: achievementID,
				State: models.GormBitSet{
					BitSet:      *bitset.New(achievementDetails.TotalItems),
					TotalLength: achievementDetails.TotalItems,
				},
			}
		} else {
			return nil, nil, fmt.Errorf("failed to fetch ProfileAchievement for ProfileID %d, AchievementID %d: %w", profile.ID, achievementID, err)
		}
	}

	// Ensure the BitSet is initialized if it's nil (e.g. from a freshly created but unsaved pa)
	if pa.State.BitSet.Bytes() == nil {
		log.Printf("UserService: ProfileAchievement.State.BitSet is uninitialized for ProfileID %d, AchievementID %d. Initializing to %d bits.", profile.ID, achievementID, achievementDetails.TotalItems)
		pa.State.BitSet = *bitset.New(achievementDetails.TotalItems)
	}

	// --- THIS IS THE NEW LOGIC FOR RANDOM SELECTION ---
	currentBitSet := &pa.State.BitSet
	maxIndex := achievementDetails.TotalItems
	revealedCount := 0

	// Step 1: Find all available (unset) indices.
	var availableIndices []uint
	for i := uint(0); i < maxIndex; i++ {
		if !currentBitSet.Test(i) {
			availableIndices = append(availableIndices, i)
		}
	}

	if len(availableIndices) > 0 {
		// Step 2: Shuffle the list of available indices.
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(availableIndices), func(i, j int) {
			availableIndices[i], availableIndices[j] = availableIndices[j], availableIndices[i]
		})

		// Step 3: Reveal the first 'itemsToReveal' from the shuffled list.
		for _, idx := range availableIndices {
			if revealedCount >= itemsToReveal {
				break
			}
			currentBitSet.Set(idx)
			revealedCount++
		}
	}
	// --- END OF NEW LOGIC ---

	if revealedCount > 0 {
		log.Printf("UserService: Revealed %d new items for ProfileID %d, AchievementID %d.", revealedCount, profile.ID, achievementID)
		// Save the updated ProfileAchievement. This will perform an INSERT or UPDATE as needed.
		if errSave := s.db.Save(&pa).Error; errSave != nil {
			return nil, nil, fmt.Errorf("failed to save ProfileAchievement for ProfileID %d, AchievementID %d: %w", profile.ID, achievementID, errSave)
		}
		// --- THIS IS THE FIX ---
		// Step 2: Immediately reload the record from the database.
		// This ensures the returned object is an exact representation of what was persisted.
		log.Printf("UserService: Re-reading ProfileAchievement (ID: %d) from DB to ensure consistency.", pa.ID)
		if errReload := s.db.First(&pa, pa.ID).Error; errReload != nil {
			// This would be a critical error, meaning we saved but can't read it back.
			return nil, nil, fmt.Errorf("failed to reload ProfileAchievement after saving: %w", errReload)
		}
		// --- END OF FIX ---
	} else {
		log.Printf("UserService: No new items to reveal or achievement already fully revealed for ProfileID %d, AchievementID %d.", profile.ID, achievementID)
	}

	return &pa, &pa.State.BitSet, nil
}

// CompleteAchievement marks a progressive achievement as fully complete.
func (s *Service) CompleteAchievement(userID uint, achievementID uint) (*models.ProfileAchievement, *bitset.BitSet, error) {
	if userID == 0 || achievementID == 0 {
		return nil, nil, fmt.Errorf("%w: userID and achievementID must be positive", ErrInvalidInput)
	}

	log.Printf("UserService: Completing achievement for UserID %d, AchievementID %d", userID, achievementID)

	var profile models.Profile
	if err := s.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to fetch profile for UserID %d: %w", userID, err)
	}

	var achievementDetails models.Achievement
	if err := s.db.First(&achievementDetails, achievementID).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to fetch achievement details for AchievementID %d: %w", achievementID, err)
	}

	var pa models.ProfileAchievement
	// Use FirstOrCreate to handle cases where the user might have skipped all quizzes and finished somehow
	err := s.db.Where("profile_id = ? AND achievement_id = ?", profile.ID, achievementID).
		FirstOrCreate(&pa, models.ProfileAchievement{
			ProfileID:     profile.ID,
			AchievementID: achievementID,
			State:         models.GormBitSet{BitSet: *bitset.New(achievementDetails.TotalItems)},
		}).Error

	if err != nil {
		return nil, nil, fmt.Errorf("failed to find/create ProfileAchievement: %w", err)
	}

	// Set all bits to 1
	currentBitSet := &pa.State.BitSet
	maxIndex := achievementDetails.TotalItems
	for i := uint(0); i < maxIndex; i++ {
		currentBitSet.Set(i)
	}

	if errSave := s.db.Save(&pa).Error; errSave != nil {
		return nil, nil, fmt.Errorf("failed to save completed ProfileAchievement: %w", errSave)
	}

	log.Printf("UserService: Successfully completed all %d items for ProfileID %d, AchievementID %d.", maxIndex, pa.ProfileID, achievementID)
	return &pa, &pa.State.BitSet, nil
}

func (s *Service) GetAllAchievements() ([]models.Achievement, error) {
	log.Println("UserService: GetAllAchievements called.")
	var achievements []models.Achievement
	if err := s.db.Order("id asc").Find(&achievements).Error; err != nil {
		log.Printf("UserService: Error fetching all achievements: %v", err)
		return nil, fmt.Errorf("could not fetch achievements from database: %w", err)
	}
	return achievements, nil
}
