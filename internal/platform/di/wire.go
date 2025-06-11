//go:build wireinject
// +build wireinject

package di

import (
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/imagegen"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/persistence/postgres"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/callback"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/command"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/message"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/achievement"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	domainUser "github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
	achCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/achievement"
	courseCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/course"
	quizCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/quiz"
	userCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/user"
	achQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/achievement"
	courseQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/course"
	userQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/user"
	"github.com/google/wire"
	"gorm.io/gorm"
)

// App contains the application's resolved dependencies.
type App struct {
	CommandHandler  *command.Handler
	MessageHandler  *message.Handler
	CallbackHandler *callback.Handler
}

// userSet provides all dependencies related to the user domain.
var userSet = wire.NewSet(
	postgres.NewUserRepository,
	wire.Bind(new(domainUser.Repository), new(*postgres.UserRepository)),
	userCmd.NewRegisterUserHandler,
	userQueries.NewGetProfileHandler,
)

// courseSet provides all dependencies related to the course domain.
var courseSet = wire.NewSet(
	postgres.NewCourseRepository,
	wire.Bind(new(course.Repository), new(*postgres.CourseRepository)),
	courseQueries.NewListCoursesHandler,
	courseQueries.NewGetOverviewHandler,
	courseCmd.NewStartSessionHandler,
	courseCmd.NewAdvanceWordHandler,
)

// wordSet provides all dependencies related to the word domain.
var wordSet = wire.NewSet(
	postgres.NewWordRepository,
	wire.Bind(new(word.Repository), new(*postgres.WordRepository)),
)

// quizSet provides all dependencies related to the quiz domain.
var quizSet = wire.NewSet(
	postgres.NewQuizRepository,
	wire.Bind(new(quiz.Repository), new(*postgres.QuizRepository)),
	quizCmd.NewCreateCourseQuizHandler,
	quizCmd.NewCreateReviewQuizHandler,
	quizCmd.NewSubmitAnswerHandler,
)
var achievementSet = wire.NewSet(
	postgres.NewAchievementRepository,
	wire.Bind(new(achievement.Repository), new(*postgres.AchievementRepository)),
	achCmd.NewAwardProgressHandler,
	achQueries.NewGetAllHandler,
)

var imagegenSet = wire.NewSet(
	wire.Value("tmp/achievements"), // Providing the output dir as a value
	imagegen.NewGenerator,
	wire.Bind(new(achievement.ImageGenerator), new(*imagegen.Generator)),
)

// InitializeApp creates the dependency graph for the application.
func InitializeApp(db *gorm.DB) (*App, error) {
	wire.Build(
		userSet,
		courseSet,
		wordSet,
		quizSet,
		achievementSet,
		imagegenSet,
		command.NewHandler,
		message.NewHandler,
		callback.NewHandler,
		wire.Struct(new(App), "*"),
	)
	return nil, nil
}
