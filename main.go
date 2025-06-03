package main

import (
	"log"
	"os" // For TEL_BOT_TOKEN, DB vars if not already handled by godotenv elsewhere

	"github.com/2000ostd/enssi-tel-bot/internal/bot"
	"github.com/2000ostd/enssi-tel-bot/internal/db"
	"github.com/2000ostd/enssi-tel-bot/internal/services"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file, relying on environment variables.")
	}

	dbConnection, err := db.ConnectDB()
	if err != nil {
		log.Fatalf("FATAL: Could not initialize database connection: %v", err)
	}
	log.Println("Database connection successful.")

	appServices, err := services.NewAppServices(dbConnection)
	if err != nil {
		log.Fatalf("FATAL: Could not initialize application services: %v", err)
	}
	log.Println("Application services initialized successfully.")

	// Assuming TEL_BOT_TOKEN is loaded from .env or environment
	token := os.Getenv("TEL_BOT_TOKEN")
	if token == "" {
		log.Fatal("FATAL: TEL_BOT_TOKEN environment variable not set.")
	}

	botInstance, err := bot.InitializeBot(token, appServices)
	if err != nil {
		log.Fatalf("FATAL: Could not initialize bot: %v", err)
	}
	log.Println("Bot initialization successful.")

	log.Println("Starting bot...")
	botInstance.Start()
}

