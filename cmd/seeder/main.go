package main

import (
	"log"

	"github.com/bazarghan/enssi-tel-bot/internal/platform/config"
	"github.com/bazarghan/enssi-tel-bot/internal/platform/database"
	"github.com/bazarghan/enssi-tel-bot/internal/seeder"
)

func main() {
	log.Println("INFO: Seeder starting...")

	// 1. Load main application configuration to get DB credentials
	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		log.Fatalf("FATAL: Could not load configuration: %v", err)
	}

	// 2. Initialize database connection
	db, err := database.ConnectDB(cfg.Database)
	if err != nil {
		log.Fatalf("FATAL: Could not initialize database connection: %v", err)
	}
	log.Println("INFO: Database connection successful.")

	// 3. Create and run the seeder
	appSeeder := seeder.NewSeeder(db)
	if err := appSeeder.Run(); err != nil {
		log.Fatalf("FATAL: Seeder failed: %v", err)
	}

	log.Println("INFO: Seeder finished successfully.")
}
