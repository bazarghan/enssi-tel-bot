package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bazarghan/enssi-tel-bot/internal/platform/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	migrationsRootFolder = "migrations"
)

func main() {
	// 1. Load configuration
	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		log.Fatalf("FATAL: could not load configuration: %v", err)
	}

	// 2. Build DSN (Database Source Name) for the migration tool
	// Note: The driver requires the `postgres` scheme, not `postgresql`.
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	// 3. Get the command from the arguments
	if len(os.Args) < 2 {
		log.Fatal("FATAL: missing command. Usage: go run ./cmd/migrate <command> [args]")
	}
	command := os.Args[1]

	// 4. Execute the command
	switch command {
	case "create":
		if len(os.Args) < 3 {
			log.Fatal("FATAL: missing migration name. Usage: go run ./cmd/migrate create <migration_name>")
		}
		createMigrationFiles(os.Args[2])
	case "up", "down":
		runMigrations(dsn, command)
	default:
		log.Fatalf("FATAL: unknown command '%s'. Available commands: create, up, down", command)
	}
}

// runMigrations applies or rolls back migrations.
func runMigrations(dsn, command string) {
	m, err := migrate.New("file://"+migrationsRootFolder, dsn)
	if err != nil {
		log.Fatalf("FATAL: failed to init migrate instance: %v", err)
	}

	var action string
	var actionErr error

	if command == "up" {
		action = "Applying"
		actionErr = m.Up()
	} else if command == "down" {
		action = "Reverting"
		actionErr = m.Down()
	}

	log.Printf("Action: %s migrations...", action)

	if actionErr != nil && !errors.Is(actionErr, migrate.ErrNoChange) {
		log.Fatalf("FATAL: failed to %s migrations: %v", command, actionErr)
	}

	if errors.Is(actionErr, migrate.ErrNoChange) {
		log.Println("INFO: No new migrations to apply.")
	} else {
		log.Println("INFO: Migrations finished successfully.")
	}
}

// createMigrationFiles creates new up and down migration files.
func createMigrationFiles(name string) {
	timestamp := time.Now().UTC().Format("20060102150405")
	basePath := fmt.Sprintf("%s/%s_%s", migrationsRootFolder, timestamp, name)

	upFile, err := os.Create(basePath + ".up.sql")
	if err != nil {
		log.Fatalf("FATAL: failed to create up migration file: %v", err)
	}
	defer upFile.Close()

	downFile, err := os.Create(basePath + ".down.sql")
	if err != nil {
		log.Fatalf("FATAL: failed to create down migration file: %v", err)
	}
	defer downFile.Close()

	log.Printf("INFO: Created migration files: \n%s.up.sql\n%s.down.sql", basePath, basePath)
}

