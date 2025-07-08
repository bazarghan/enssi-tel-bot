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
	@echo "  run-bot           Run the telegram bot locally"
	@echo "  run-worker        Run the background worker locally"
	@echo "  build             Build both bot and worker binaries"
	@echo "  test              Run all go tests"
	@echo "  wire              Generate wire dependency injection files"
	@echo "  docker-build      Build the docker images using docker-compose"
	@echo "  docker-up         Start all services with docker-compose"
	@echo "  docker-down       Stop all services with docker-compose"
	@echo "  clean             Remove build artifacts and test cache"


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

