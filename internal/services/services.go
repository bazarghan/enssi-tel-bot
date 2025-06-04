package services

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/services/course"
	"github.com/2000ostd/enssi-tel-bot/internal/services/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/services/user"
	"github.com/2000ostd/enssi-tel-bot/internal/services/word"
	"gorm.io/gorm"
)

// AppServices provides access to all application services.
type AppServices struct {
	userServiceInstance   user.Service
	courseServiceInstance course.Service
	quizServiceInstance   quiz.Service
	wordServiceInstance   word.Service
}

// NewAppServices creates and initializes all application services and their dependencies.
func NewAppServices(db *gorm.DB) (*AppServices, error) {
	if db == nil {
		return nil, fmt.Errorf("database instance is required to initialize AppServices")
	}

	userService := user.NewService(db)
	wordService := word.NewService(db)

	// Quiz Service now depends on WordService
	quizService := quiz.NewService(db, wordService) // Pass wordService instance

	// Course Service depends on QuizService and WordService
	courseService := course.NewService(db, quizService, wordService)

	return &AppServices{
		userServiceInstance:   *userService,
		courseServiceInstance: *courseService,
		quizServiceInstance:   *quizService,
		wordServiceInstance:   *wordService,
	}, nil
}

// User returns the user service interface.
func (as *AppServices) User() user.UserService {
	return &as.userServiceInstance
}

// Course returns the course service interface.
func (as *AppServices) Course() course.CourseService {
	return &as.courseServiceInstance
}

// Quiz returns the quiz service interface.
func (as *AppServices) Quiz() quiz.QuizService {
	return &as.quizServiceInstance
}

// Word returns the word service interface.
func (as *AppServices) Word() word.WordService {
	return &as.wordServiceInstance
}

