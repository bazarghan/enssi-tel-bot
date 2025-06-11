package postgres

import "gorm.io/gorm"

// courseModel is the GORM-specific struct for the 'courses' table.
type courseModel struct {
	gorm.Model
	Title               string `gorm:"not null;check:length(title) >= 2"`
	PersianTitle        string `gorm:"not null;check:length(title) >= 2"`
	Description         string
	PersianDescription  string
	LinkedAchievementID uint `gorm:"default:0"`
}

func (courseModel) TableName() string {
	return "courses"
}

// userCourseModel is the GORM-specific struct for the 'user_courses' join table.
type userCourseModel struct {
	gorm.Model
	UserID   uint
	CourseID uint
	Progress uint
}

func (userCourseModel) TableName() string {
	return "user_courses"
}

// courseWordModel is a temporary model to count words before the Word domain is migrated.
type courseWordModel struct {
	CourseID uint
	WordID   uint
}

func (courseWordModel) TableName() string {
	return "course_words"
}
