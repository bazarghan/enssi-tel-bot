package course

import (
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"log"

	"github.com/bits-and-blooms/bitset" // Import for bitset
)

func (s *Service) populateAchievementInfo(
	linkedAchievementID uint,
	newBitsetState *bitset.BitSet,
) *AchievementUpdateInfo {

	var achievementInfo *AchievementUpdateInfo

	var achievementDetails models.Achievement
	if errAchDetails := s.db.First(&achievementDetails, linkedAchievementID).Error; errAchDetails == nil {
		log.Printf("CourseService: Achievement progress awarded. Populating DTO for handler.")
		achievementInfo = &AchievementUpdateInfo{
			AchievementID:  linkedAchievementID,
			Title:          achievementDetails.Title,
			ImageURL:       achievementDetails.ImageURL,
			TotalItems:     achievementDetails.TotalItems,
			GridWidth:      achievementDetails.GridWidth,
			GridHeight:     achievementDetails.GridHeight,
			NewStateBitSet: newBitsetState,
		}
	}
	return achievementInfo

}
