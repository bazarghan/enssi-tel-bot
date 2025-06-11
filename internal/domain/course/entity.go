package course

// Course represents the core properties of a learning course.
type Course struct {
	ID                  uint
	Title               string
	PersianTitle        string
	Description         string
	PersianDescription  string
	TotalWords          int
	LinkedAchievementID uint
}

// UserProgress represents a user's progress within a single course.
type UserProgress struct {
	ProgressPercentage int
	WordsCompleted     int
	IsCompleted        bool
	IsStarted          bool
}
