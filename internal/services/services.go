// internal/services/services.go
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
// It acts as a container and helps in managing service dependencies during initialization.
type AppServices struct {
	// Store concrete service implementations internally.
	// Handlers will access them via interface-returning getter methods.
	userServiceInstance   user.Service   // Using concrete type for internal storage
	courseServiceInstance course.Service // Using concrete type
	quizServiceInstance   quiz.Service   // Using concrete type
	wordServiceInstance   word.Service   // Using concrete type
}

// NewAppServices creates and initializes all application services and their dependencies.
// This is where you "wire up" your application's service layer.
func NewAppServices(db *gorm.DB) (*AppServices, error) {
	if db == nil {
		return nil, fmt.Errorf("database instance is required to initialize AppServices")
	}

	// Initialize services in an order that respects dependencies.
	// Services that don't depend on other application services can be initialized first.

	// User Service (typically no app service dependencies, only DB)
	userService := user.NewService(db)

	// Word Service (typically no app service dependencies, only DB)
	wordService := word.NewService(db)

	// Quiz Service (no longer depends on CourseService for this iteration)
	quizService := quiz.NewService(db)

	// Course Service depends on QuizService and WordService (passing the concrete instances,
	// as the NewService functions for course, quiz etc. accept their interface types)
	courseService := course.NewService(db, quizService, wordService)

	return &AppServices{
		userServiceInstance:   *userService,   // Store the value
		courseServiceInstance: *courseService, // Store the value
		quizServiceInstance:   *quizService,   // Store the value
		wordServiceInstance:   *wordService,   // Store the value
	}, nil
}

// --- Accessor methods for services (returning interfaces) ---
// These methods allow handlers or other parts of your application to get
// instances of services, respecting their defined interfaces.

// User returns the user service interface.
func (as *AppServices) User() user.UserService {
	return &as.userServiceInstance // Return a pointer to the stored service struct
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

