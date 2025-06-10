#!/usr/bin/env bash
set -euo pipefail

##############################################################################
# root-level boilerplate
##############################################################################
touch .env .gitignore Dockerfile Makefile README.md tools.go
touch go.mod go.sum                         # you’ll overwrite these later

##############################################################################
# static assets
##############################################################################
mkdir -p assets/imgs/achievements
# (real PNGs will live here; keep directory committed)
touch assets/imgs/achievements/.gitkeep

##############################################################################
# top-level config file
##############################################################################
mkdir -p configs
touch configs/config.yaml

##############################################################################
# binaries
##############################################################################
mkdir -p cmd/{bot,migrate,worker}
touch cmd/bot/main.go cmd/migrate/main.go cmd/worker/main.go

##############################################################################
# platform layer
##############################################################################
mkdir -p internal/platform/{config,database,di,observability/logger,observability/metrics}
touch internal/platform/config/{loader.go,types.go}
touch internal/platform/database/{postgres.go,transactor.go}
touch internal/platform/di/wire.go
touch internal/platform/observability/logger/logger.go
touch internal/platform/observability/metrics/{metrics.go,noop.go}

##############################################################################
# domain layer
##############################################################################
mkdir -p internal/domain/{user,course}
touch internal/domain/user/{entity.go,errors.go,repository.go}
touch internal/domain/course/{entity.go,errors.go,repository.go}

##############################################################################
# use-case layer
##############################################################################
mkdir -p internal/usecase/{commands/user,queries/user}
touch internal/usecase/commands/user/register_user.go
touch internal/usecase/queries/user/get_profile.go

##############################################################################
# adapters – Postgres persistence
##############################################################################
mkdir -p internal/adapter/persistence/postgres
touch internal/adapter/persistence/postgres/{models.go,mapper.go,user_repo.go,course_repo.go}
touch internal/adapter/persistence/postgres/user_repo_contract_test.go
touch internal/adapter/persistence/postgres/course_repo_contract_test.go

##############################################################################
# adapters – Telegram
##############################################################################
mkdir -p internal/adapter/telegram/{dto,handlers/{callback,command,message}}
touch internal/adapter/telegram/{bot.go,router.go,middleware.go}
touch internal/adapter/telegram/dto/{user.go,course.go}
touch internal/adapter/telegram/handlers/callback/handler.go
touch internal/adapter/telegram/handlers/command/handler.go
touch internal/adapter/telegram/handlers/message/handler.go

##############################################################################
# adapters – Image generator
##############################################################################
mkdir -p internal/adapter/imagegen
touch internal/adapter/imagegen/generator.go

##############################################################################
# migrations
##############################################################################
mkdir -p internal/migrations
touch internal/migrations/0001_initial_schema.sql

##############################################################################
# pkg helpers
##############################################################################
mkdir -p pkg/{imagekit,tgmarkdown}
touch pkg/imagekit/mosaic.go
touch pkg/tgmarkdown/escaper.go

##############################################################################
# integration-test scaffold
##############################################################################
mkdir -p test/integration
touch test/integration/main_test.go

echo "✅  Skeleton created – commit when ready."
