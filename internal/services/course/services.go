package course

import (
	"github.com/2000ostd/enssi-tel-bot/internal/services/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/services/word"

	"github.com/2000ostd/enssi-tel-bot/internal/services/user" // Import user service
	"gorm.io/gorm"
)

//==============================================================================
//  Service Definition & Initialization
//==============================================================================

// Service implements the CourseService interface.
type Service struct {
	db          *gorm.DB
	quizService quiz.QuizService
	wordService word.WordService
	userService user.UserService
}

// NewService creates a new instance of the course Service.
func NewService(db *gorm.DB, qs quiz.QuizService, ws word.WordService, us user.UserService) *Service {
	return &Service{
		db:          db,
		quizService: qs,
		wordService: ws,
		userService: us,
	}
}

// Ensure Service implements CourseService interface
var _ CourseService = (*Service)(nil)
