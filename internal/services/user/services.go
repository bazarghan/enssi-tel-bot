package user

import (
	"errors"
	"fmt"
	"log"

	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"github.com/bits-and-blooms/bitset" // Import for bitset
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

// Service implements the UserService interface.
// ... (existing NewService and other methods) ...

// AwardAchievementProgress updates the state of a progressive achievement for a user.
// It reveals a specified number of items (e.g., pixels or blocks).
// It finds the next available 'itemsToReveal' number of unset bits and sets them.
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

	var pa models.ProfileAchievement
	err := s.db.Where("profile_id = ? AND achievement_id = ?", profile.ID, achievementID).
		First(&pa).Error

	var achievementDetails models.Achievement
	if errAch := s.db.First(&achievementDetails, achievementID).Error; errAch != nil {
		log.Printf("UserService: Failed to fetch achievement details for AchievementID %d: %v", achievementID, errAch)
		// Decide if this is critical. For now, let's assume we need total pixels from it.
		// If TotalPixels, GridWidth etc. are on the achievement model, this is where you'd use them.
		// For this example, let's assume 504 total pixels for the specific achievement.
		// This should ideally come from achievementDetails.TotalPixels if that field exists.
	}
	// Placeholder total pixels - this should ideally come from the achievementDetails.TotalPixels
	// or be a known constant for this specific achievement type.
	const totalPixelBlocksFor504Achievement = 504

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("UserService: No existing ProfileAchievement for ProfileID %d, AchievementID %d. Creating new one.", profile.ID, achievementID)
			// Initialize a new bitset. Its length should correspond to the total items for this achievement.
			// This length needs to be known for the specific achievement.
			newState := bitset.New(totalPixelBlocksFor504Achievement) // e.g., 504 for the 504 words achievement

			pa = models.ProfileAchievement{
				ProfileID:     profile.ID,
				AchievementID: achievementID,
				State:         models.GormBitSet{BitSet: *newState},
			}
			// The create will be part of the transaction below if we modify it.
			// For now, let's assume we create then update, or handle within transaction.
		} else {
			return nil, nil, fmt.Errorf("failed to fetch ProfileAchievement for ProfileID %d, AchievementID %d: %w", profile.ID, achievementID, err)
		}
	}

	// Ensure the BitSet is initialized if it's nil (e.g. from a freshly created but unsaved pa)
	// The GormBitSet.Scan method should handle initializing an empty BitSet if the DB field was empty.
	// However, if pa was just created in memory above, pa.State.BitSet might be the zero value.
	if pa.State.BitSet.Bytes() == nil { // Check if the underlying bitset is uninitialized (e.g. length 0 or nil buffer)
		log.Printf("UserService: ProfileAchievement.State.BitSet is uninitialized for ProfileID %d, AchievementID %d. Initializing to %d bits.", profile.ID, achievementID, totalPixelBlocksFor504Achievement)
		pa.State.BitSet = *bitset.New(totalPixelBlocksFor504Achievement)
	}

	revealedCount := 0
	// Iterate from the first bit to find unrevealed pixels.
	// The length of the bitset should be fixed for a given achievement (e.g., 504).
	// We need to ensure pa.State.BitSet is properly sized if it's new or was empty.
	// bitset.New(N) creates a bitset that can hold indices up to N-1.
	// A bitset for 504 pixels should be bitset.New(504).

	// Ensure the bitset has the correct capacity if it's being used for the first time
	// or if its current length is less than what's expected for this achievement.
	// This is a bit tricky as GormBitSet doesn't directly expose a resize.
	// When a new GormBitSet is created, it should be with `bitset.New(totalPixelBlocksFor504Achievement)`.
	// When loading from DB, `Scan` populates it. If Scan results in an empty/small bitset from an empty DB field,
	// we need to handle its conceptual "full size".

	currentBitSet := &pa.State.BitSet // Get a pointer to the bitset to modify it

	// The length of the conceptual bitset for this achievement.
	// Should be obtained from achievementDetails.TotalPixels if available.
	maxIndex := uint(totalPixelBlocksFor504Achievement)

	for i := uint(0); i < maxIndex && revealedCount < itemsToReveal; i++ {
		if !currentBitSet.Test(i) { // If pixel 'i' is not yet revealed
			currentBitSet.Set(i) // Reveal it
			revealedCount++
		}
	}

	if revealedCount > 0 {
		log.Printf("UserService: Revealed %d new items for ProfileID %d, AchievementID %d.", revealedCount, profile.ID, achievementID)
		// Save the updated ProfileAchievement
		// If pa was newly created in this function, this will perform an insert.
		// If pa was fetched, this will perform an update.
		if errSave := s.db.Save(&pa).Error; errSave != nil {
			return nil, nil, fmt.Errorf("failed to save ProfileAchievement for ProfileID %d, AchievementID %d: %w", profile.ID, achievementID, errSave)
		}
	} else {
		log.Printf("UserService: No new items to reveal or achievement already fully revealed for ProfileID %d, AchievementID %d.", profile.ID, achievementID)
	}

	return &pa, &pa.State.BitSet, nil
}

// ... (rest of your user service methods: GetUserProfile, etc.) ...
