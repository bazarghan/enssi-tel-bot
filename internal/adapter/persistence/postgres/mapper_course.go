package postgres

import "github.com/bazarghan/enssi-tel-bot/internal/domain/course"

// toDomainCourse converts a GORM courseModel to a domain Course entity.
func toDomainCourse(m CourseModel, totalWords int) course.Course {
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
