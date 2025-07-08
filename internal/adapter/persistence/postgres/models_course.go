package postgres

import "gorm.io/gorm"

// courseModel is the GORM-specific struct for the 'courses' table.
type CourseModel struct {
	gorm.Model
	Title               string `gorm:"not null;check:length(title) >= 2"`
	PersianTitle        string `gorm:"not null;check:length(title) >= 2"`
	Description         string
	PersianDescription  string
	LinkedAchievementID uint `gorm:"default:0"`
}

func (CourseModel) TableName() string {
	return "courses"
}

// userCourseModel is the GORM-specific struct for the 'user_courses' join table.
type UserCourseModel struct {
	gorm.Model
	UserID   uint
	CourseID uint
	Progress uint
}

func (UserCourseModel) TableName() string {
	return "user_courses"
}

// courseWordModel is a temporary model to count words before the Word domain is migrated.
type CourseWordModel struct {
	gorm.Model        // Required for ID field
	CourseID          uint
	WordID            uint
	TelgramImageID    string `gorm:"not null`
	TelgramImageDocID string `gorm:"not null`
	Lesson            string
	Index             uint
}

func (CourseWordModel) TableName() string {
	return "course_words"
}
