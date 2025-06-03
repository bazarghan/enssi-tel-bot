package course

import (
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"github.com/2000ostd/enssi-tel-bot/internal/services/quiz"
)

// CourseService defines the interface for course-related operations.
type CourseService interface {
	// ListAvailableCourses retrieves all courses a user can take, possibly with their progress.
	ListAvailableCourses(userID uint) ([]CourseSummaryView, error)

	// GetCourseOverView provides details for a specific course, including user's overall progress.
	GetCourseOverview(courseID uint, userID uint) (*CourseOverview, error)

	// StartOrResumeLearningSession begins or continues a learning session in a course.
	// It determines if a quiz is due or what word to present.
	StartOrResumeLearningSession(courseID uint, userID uint) (*LearningContext, error)

	// AdvanceToNextWord gets the next word in the learning sequence for a course,
	// potentially advancing progress and checking for quiz triggers.
	AdvanceToNextWord(courseID uint, userID uint) (*LearningContext, error)

	HandleQuizCompletion(userID uint, courseID uint, quizOutcome *quiz.QuizResult) (*LearningContext, error)
	// UpdateUserCourseProgress is a lower-level method to directly set a user's progress in a course.
	// This might be used internally or by other services (e.g., QuizService after a quiz).
	UpdateUserCourseProgress(userID uint, courseID uint, newProgress uint) (*models.UserCourse, error)

	// GetUserCourse retrieves the UserCourse record directly.
	GetUserCourse(userID uint, courseID uint) (*models.UserCourse, error)
}
