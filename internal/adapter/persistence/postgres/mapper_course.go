package postgres

import "github.com/2000ostd/enssi-tel-bot/internal/domain/course"

// toDomainCourse converts a GORM courseModel to a domain Course entity.
func toDomainCourse(m courseModel, totalWords int) course.Course {
	return course.Course{
		ID:                  m.ID,
		Title:               m.Title,
		PersianTitle:        m.PersianTitle,
		Description:         m.Description,
		PersianDescription:  m.PersianDescription,
		TotalWords:          totalWords,
		LinkedAchievementID: m.LinkedAchievementID,
	}
}
