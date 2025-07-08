# Go parameters
BOT_BINARY=enssi-bot
WORKER_BINARY=enssi-worker
BOT_CMD_PATH=./cmd/bot
WORKER_CMD_PATH=./cmd/worker

.PHONY: help run-bot run-worker build test wire clean docker-build docker-up docker-down

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  --- Local Development ---"
	@echo "  run-bot         Run the telegram bot locally"
	@echo "  run-worker      Run the background worker locally"
	@echo "  build           Build both bot and worker binaries"
	@echo "  test            Run all go tests"
	@echo "  wire            Generate wire dependency injection files"
	@echo "  clean           Remove build artifacts and test cache"
	@echo ""
	@echo "  --- Database ---"
	@echo "  db-reset        ⚠️  Resets the dev database (drops tables, migrates, and seeds)"
	@echo "  seed            Populate the database with initial data (courses, words, etc.)"
	@echo "  migrate-up      Apply all available 'up' database migrations"
	@echo "  migrate-down    Revert the last 'down' database migration"
	@echo "  migrate-create  Create new migration files. Usage: make migrate-create name=<migration_name>"
	@echo ""
	@echo "  --- Docker ---"
	@echo "  docker-build    Build the docker images using docker-compose"
	@echo "  docker-up       Start all services with docker-compose"
	@echo "  docker-down     Stop all services with docker-compose"
# --- Local Development Targets ---

run-bot:
	@echo "Running bot..."
	@go run $(BOT_CMD_PATH)/main.go

run-worker:
	@echo "Running worker..."
	@go run $(WORKER_CMD_PATH)/main.go

build:
	@echo "Building binaries..."
	@go build -o bin/$(BOT_BINARY) $(BOT_CMD_PATH)
	@go build -o bin/$(WORKER_BINARY) $(WORKER_CMD_PATH)
	@echo "Build complete."

test:
	@echo "Running tests..."
	@go test -v ./...

wire:
	@echo "Generating wire files..."
	@cd internal/platform/di && go generate

clean:
	@echo "Cleaning up..."
	@go clean -testcache
	@rm -f bin/$(BOT_BINARY) bin/$(WORKER_BINARY)


# --- Docker Targets ---

docker-build:
	@echo "Building docker images..."
	@docker-compose build

docker-up:
	@echo "Starting services with docker-compose..."
	@docker-compose up -d

docker-down:
	@echo "Stopping services with docker-compose..."
	@docker-compose down


# --- Database Migration Targets ---

## Creates new up/down migration files. Usage: make migrate-create name=add_new_feature
migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Usage: make migrate-create name=<migration_name>"; \
		exit 1; \
	fi
	@echo "Creating migration files for: $(name)..."
	@go run ./cmd/migrate create $(name)

## Applies all available 'up' migrations.
migrate-up:
	@echo "Running 'up' migrations..."
	@go run ./cmd/migrate up

## Reverts the last 'down' migration.
migrate-down:
	@echo "Running 'down' migration..."
	@go run ./cmd/migrate down


# --- Database Seeding Target ---

## Populates the database with initial data (achievements, courses, words).
seed:
	@echo "Running database seeder..."
	@go run ./cmd/seeder/main.go


# --- Full Database Reset (DEVELOPMENT ONLY) ---

## WARNING: Destructive! Drops all tables, re-migrates, and re-seeds the database.
## Do NOT run this on a production database.
db-reset:
	@echo "⚠️  WARNING: This will drop all tables, re-migrate, and re-seed the database."
	@read -p "Are you sure you want to continue? (y/n) " -r response; \
	if [ "$$response" = "y" ] || [ "$$response" = "Y" ]; then \
		echo "\nProceeding with database reset..."; \
		$(MAKE) migrate-down; \
		$(MAKE) migrate-up; \
		$(MAKE) seed; \
		echo "✅  Database reset complete."; \
	else \
		echo "\nDatabase reset aborted."; \
	fi

