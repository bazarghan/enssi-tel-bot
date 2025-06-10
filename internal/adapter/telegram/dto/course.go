package dto

// CourseSummary is a DTO for displaying a course in a list.
type CourseSummary struct {
	ID                 uint
	PersianTitle       string
	ProgressPercentage int
	IsCompleted        bool
}

// CourseOverview is a DTO for displaying a course's detailed view.
type CourseOverview struct {
	ID                     uint
	Title                  string
	PersianTitle           string
	PersianFullDescription string
	TotalWords             int
	ProgressPercentage     int
	IsCompleted            bool
	IsStarted              bool
}
