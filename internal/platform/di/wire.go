//go:generate wire
//go:build wireinject
// +build wireinject

package di

import (
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/imagegen"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/persistence/postgres"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/callback"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/command"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/message"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/achievement"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/notification"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	domainUser "github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
	achCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/achievement"
	courseCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/course"
	quizCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/quiz"
	userCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/user"
	"github.com/2000ostd/enssi-tel-bot/internal/usecase/jobs"
	achQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/achievement"
	courseQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/course"
	userQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/user"
	"github.com/google/wire"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

// BotApp contains the dependencies for the Telegram bot entry point.
type BotApp struct {
	CommandHandler  *command.Handler
	MessageHandler  *message.Handler
	CallbackHandler *callback.Handler
}

// WorkerApp contains the dependencies for the background worker entry point.
type WorkerApp struct {
	TriggerDailyReviewsJob *jobs.TriggerDailyReviewsJob
}

var userSet = wire.NewSet(
	postgres.NewUserRepository,
	wire.Bind(new(domainUser.Repository), new(*postgres.UserRepository)),
	userCmd.NewRegisterUserHandler,
	userQueries.NewGetProfileHandler,
)

var courseSet = wire.NewSet(
	postgres.NewCourseRepository,
	wire.Bind(new(course.Repository), new(*postgres.CourseRepository)),
	courseQueries.NewListCoursesHandler,
	courseQueries.NewGetOverviewHandler,
	courseCmd.NewStartSessionHandler,
	courseCmd.NewAdvanceWordHandler,
	courseCmd.NewHandleQuizCompletionHandler,
)

var wordSet = wire.NewSet(
	postgres.NewWordRepository,
	wire.Bind(new(word.Repository), new(*postgres.WordRepository)),
)

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
	wire.Value("assets/imgs/achievements_gen"), // Using a sub-directory for generated images
	imagegen.NewGenerator,
	wire.Bind(new(achievement.ImageGenerator), new(*imagegen.Generator)),
)

var notifierSet = wire.NewSet(
	telegram.NewNotifier,
	wire.Bind(new(notification.Notifier), new(*telegram.Notifier)),
)

var wordCacheSet = wire.NewSet(
	wordCmd.NewCacheMediaHandler,
)

// InitializeBotApp creates the dependency graph for the bot application handlers.
func InitializeBotApp(db *gorm.DB) (*BotApp, error) {
	wire.Build(
		userSet,
		courseSet,
		wordSet,
		wordCacheSet, // Add the new set
		quizSet,
		achievementSet,
		imagegenSet,
		command.NewHandler,
		message.NewHandler,
		callback.NewHandler,
		wire.Struct(new(BotApp), "*"),
	)
	return nil, nil // This return is a placeholder for Wire
}

// InitializeWorkerApp creates the dependency graph for the worker application.
func InitializeWorkerApp(db *gorm.DB, bot *telebot.Bot) (*WorkerApp, error) {
	wire.Build(
		userSet,
		wordSet,
		quizSet,
		notifierSet,
		jobs.NewTriggerDailyReviewsJob,
		wire.Struct(new(WorkerApp), "*"),
	)
	return nil, nil // This return is a placeholder for Wire
}

// InitializeRegisterUserHandler is a helper to get just the user registration use case.
// This is a temporary solution to simplify wiring the middleware in main.go.
func InitializeRegisterUserHandler(db *gorm.DB) userCmd.RegisterUserHandler {
	wire.Build(
		userSet,
	)
	return userCmd.RegisterUserHandler{} // This return is a placeholder for Wire
}
