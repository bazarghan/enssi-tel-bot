//go:build wireinject
// +build wireinject

package di

import (
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/persistence/postgres"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/handlers/command"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	registerCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/user"
	getProfileQry "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/user"
	"github.com/google/wire"
	"gorm.io/gorm"
)

// App contains the application's dependencies.
type App struct {
	CommandHandler *command.Handler
}

// The primary provider set for the user domain.
var userSet = wire.NewSet(
	postgres.NewUserRepository,
	wire.Bind(new(user.Repository), new(*postgres.UserRepository)),
	registerCmd.NewRegisterUserHandler,
	getProfileQry.NewGetProfileHandler,
)

// The provider for the top-level command handler.
var commandHandlerSet = wire.NewSet(
	userSet,
	command.NewHandler,
)

// InitializeApp creates the dependency graph for the application.
func InitializeApp(db *gorm.DB) (*App, error) {
	wire.Build(
		commandHandlerSet,
		wire.Struct(new(App), "*"),
	)
	return nil, nil
}
