//go:build wireinject
// +build wireinject

package di

import (
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/persistence/postgres"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/command"
	registerCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/user"
	getProfileQry "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/user"
	"github.com/google/wire"
	"gorm.io/gorm"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/message"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	domainUser "github.com/2000ostd/enssi-tel-bot/internal/domain/user"

	getOverviewQry "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/course"
	listCoursesQry "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/course"
)

// App contains the application's dependencies.
type App struct {
	CommandHandler *command.Handler

	MessageHandler *message.Handler
}

// The primary provider set for the user domain.
var userSet = wire.NewSet(
	postgres.NewUserRepository,
	wire.Bind(new(domainUser.Repository), new(*postgres.UserRepository)),
	registerCmd.NewRegisterUserHandler,
	getProfileQry.NewGetProfileHandler,
)

// The provider for the top-level command handler.
var commandHandlerSet = wire.NewSet(
	userSet,
	command.NewHandler,
)

// The primary provider set for the course domain.
var courseSet = wire.NewSet(
	postgres.NewCourseRepository,
	wire.Bind(new(course.Repository), new(*postgres.CourseRepository)),
	listCoursesQry.NewListCoursesHandler,
	getOverviewQry.NewGetOverviewHandler,
)

// InitializeApp creates the dependency graph for the application.
func InitializeApp(db *gorm.DB) (*App, error) {
	wire.Build(
		commandHandlerSet,
		courseSet,
		message.NewHandler,
		wire.Struct(new(App), "*"),
	)
	return nil, nil
}
