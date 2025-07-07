// File: internal/platform/di/wire.go

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
	"github.com/2000ostd/enssi-tel-bot/internal/platform/config"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	achCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/achievement"
	adminCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/admin"
	courseCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/course"
	quizCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/quiz"
	userCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/user"
	wordCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/word"
	"github.com/2000ostd/enssi-tel-bot/internal/usecase/jobs"
	achQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/achievement"
	adminQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/admin"
	courseQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/course"
	reviewQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/review"
	userQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/user"
	"github.com/google/wire"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

// BotApp remains the same.
type BotApp struct {
	CommandHandler      *command.Handler
	MessageHandler      *message.Router
	CallbackHandler     *callback.Handler
	RegisterUserHandler userCmd.RegisterUserHandler
}

// WorkerApp remains the same.
type WorkerApp struct {
	TriggerDailyReviewsJob *jobs.TriggerDailyReviewsJob
}

// --- Provider Sets ---
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

var reviewSet = wire.NewSet(reviewQueries.NewHandler)

var wordSet = wire.NewSet(
	postgres.NewWordRepository,
	wire.Bind(new(word.Repository), new(*postgres.WordRepository)),
)

var statsHandlerSet = wire.NewSet(adminQueries.NewGetStatsHandler)
var broadcastHandlerSet = wire.NewSet(adminCmd.NewBroadcastHandler)

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
	achCmd.NewHandleWordMasteryHandler,
)

var imagegenSet = wire.NewSet(
	wire.Value("assets/imgs/achievements_gen"),
	imagegen.NewGenerator,
	wire.Bind(new(achievement.ImageGenerator), new(*imagegen.Generator)),
)

var notifierSet = wire.NewSet(
	telegram.NewNotifier,
	wire.Bind(new(notification.Notifier), new(*telegram.Notifier)),
)

var wordCacheSet = wire.NewSet(wordCmd.NewCacheMediaHandler)

var messageHandlerSet = wire.NewSet(
	message.NewAdminHandler,
	message.NewCourseHandler,
	message.NewProfileHandler,
	message.NewMainMenuHandler,
	message.NewRouter,
)

// --- UPDATED Injectors ---

// InitializeBotApp now gets the logger as an input.
func InitializeBotApp(cfg *config.Config, db *gorm.DB, appLogger logger.Logger) (*BotApp, error) {
	wire.Build(
		userSet,
		courseSet,
		wordSet,
		wordCacheSet,
		quizSet,
		achievementSet,

		statsHandlerSet,

		reviewSet,
		imagegenSet,

		command.NewHandler,
		messageHandlerSet,
		callback.NewHandler,
		wire.Struct(new(BotApp), "*"),
	)
	return nil, nil // Placeholder for Wire
}

// InitializeWorkerApp now gets the logger as an input.
func InitializeWorkerApp(cfg *config.Config, db *gorm.DB, bot *telebot.Bot, appLogger logger.Logger) (*WorkerApp, error) {
	wire.Build(
		userSet,
		wordSet,
		quizSet,
		notifierSet,
		jobs.NewTriggerDailyReviewsJob,
		wire.Struct(new(WorkerApp), "*"),
	)
	return nil, nil // Placeholder for Wire
}

func InitializeBroadcastHandler(cfg *config.Config, db *gorm.DB, bot *telebot.Bot, appLogger logger.Logger) (adminCmd.BroadcastHandler, error) {
	wire.Build(
		userSet,
		quizSet,
		notifierSet,
		broadcastHandlerSet,
	)
	return adminCmd.BroadcastHandler{}, nil
}
