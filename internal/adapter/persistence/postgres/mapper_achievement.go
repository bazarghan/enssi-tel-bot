package postgres

import "github.com/bazarghan/enssi-tel-bot/internal/domain/achievement"

// toDomainAchievement maps the gorm model to the domain entity.
func toDomainAchievement(m AchievementModel) achievement.Achievement {
	return achievement.Achievement{
		ID:              m.ID,
		Title:           m.Title,
		Description:     m.Description,
		Type:            m.Type,
		ImageURL:        m.ImageURL,
		MinWordRequired: m.MinWordRequired,
		TotalItems:      m.TotalItems,
		CellWidth:       m.CellWidth,
		CellHeight:      m.CellHeight,
	}
}

// toDomainUserAchievement maps the gorm model to the domain entity.
func toDomainUserAchievement(m UserAchievementModel) achievement.UserAchievement {
	return achievement.UserAchievement{
		ID:            m.ID,
		AchievementID: m.AchievementID,
		EarnedAt:      m.CreatedAt,
		State:         &m.State.BitSet,
		CompletedAt:   m.CompletedAt,
	}
}

// toPersistenceUserAchievement maps the domain entity to the gorm model for saving.
func toPersistenceUserAchievement(d achievement.UserAchievement, profileID uint, totalItems uint) UserAchievementModel {
	model := UserAchievementModel{

		ProfileID:     profileID, // Use the found profileID
		AchievementID: d.AchievementID,
		CompletedAt:   d.CompletedAt,
	}
	if d.ID != 0 {
		model.ID = d.ID // Set ID for updates
	}
	if d.State != nil {
		model.State = GormBitSet{
			BitSet:      *d.State,
			TotalLength: totalItems,
		}
	}
	return model
}
