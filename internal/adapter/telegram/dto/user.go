package dto

// UserProfile represents the data structure for displaying a user profile in Telegram.
type UserProfile struct {
	FirstName     string
	Username      string
	LastName      string
	Score         uint
	WordsStudied  int
	CoursesActive int
	// Achievements will be added in a later slice
}
