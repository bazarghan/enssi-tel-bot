package course

import (
	"errors"
	"github.com/2000ostd/enssi-tel-bot/internal/services/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/services/word"
)

// --- Service-Specific Errors ---
var (
	ErrCourseNotFound         = errors.New("course service: course not found")
	ErrUserCourseNotFound     = errors.New("course service: user progress in this course not found")
	ErrCourseCompleted        = errors.New("course service: all words in this course have been studied")
	ErrInvalidCourseData      = errors.New("course service: invalid course data provided")
	ErrQuizIntegration        = errors.New("course service: error interacting with quiz service")
	ErrWordIntegration        = errors.New("course service: error interacting with word service")
	ErrCannotAdvanceNoSession = errors.New("course service: cannot advance, no active learning session or course progress found")

	ErrProgressUpdateFailed  = errors.New("course service: failed to update user progress")
	ErrEnrollmentFailed      = errors.New("course service: failed to enroll user in course")
	ErrUserCourseQueryFailed = errors.New("course service: database query for user course failed")
	ErrTotalWordCountFailed  = errors.New("course service: failed to count the total word in course")
	ErrCourseFetchFailed     = errors.New("course service: faied to fetch courses")
)

const (
	wordsPerQuizBlock = 12 // Example value

	MsgCourseNoContent     = "این دوره محتوایی ندارد. 🤷‍♂️"
	MsgCourseCompleted     = "تبریک! 🎉 شما این دوره را با موفقیت به پایان رساندید."
	MsgCourseIsEmpty       = "این دوره خالی است. 📂"
	MsgEndOfAvailableWords = "شما به پایان کلمات موجود رسیده‌اید. 🏁"
	MsgHereIsYourNextWord  = "کلمه بعدی شما اینجاست: 👇"

	MsgFinalQuizForBlockPrompt    = "شما دوازده کلمه این بخش را کامل کرده‌اید. وقت آزمون نهایی است! 🧐"
	MsgQuizDueAfterBlockPromptFmt = "شما %d کلمه در این بخش مطالعه کرده‌اید. وقت آزمون است! 🤓"
)

// CourseSummaryView is a light representation of a course for listings.
type CourseSummaryView struct {
	ID                 uint
	Title              string
	PersianTitle       string
	ShortDescription   string
	TotalWords         int64
	UserProgressWords  uint
	ProgressPercentage int
	IsCompletedByUser  bool
}

// CourseOverview provides detailed information about a course for a specific user.
type CourseOverview struct {
	ID                     uint
	Title                  string
	PersianTitle           string
	FullDescription        string
	PersianFullDescription string
	TotalWords             int64
	UserProgressWords      uint
	ProgressPercentage     int
	IsCompletedByUser      bool
}

// LearningContext represents the current state when a user is actively learning in a course.
// This tells the handler what to display: a word or a quiz.
type LearningContext struct {
	UserID        uint
	CourseID      uint
	UserProgress  uint // Current word index user is at (or has just completed if quiz is next)
	IsQuizDue     bool
	QuizState     *quiz.QuizState       // Populated if IsQuizDue is true
	WordToDisplay *word.WordDisplayData // Populated if IsQuizDue is false and there's a word
	MessageToUser string                // e.g., "Here is your next word", "Time for a quiz!"
	IsCourseEnded bool                  // True if all words + final quiz (if any) are done.
}
