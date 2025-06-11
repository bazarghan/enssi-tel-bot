package main

import (
	"log"
	"os"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/database" // CORRECTED IMPORT
	"github.com/2000ostd/enssi-tel-bot/internal/platform/di"
	"github.com/joho/godotenv"
)

func main() {
	// Load configuration
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, relying on environment variables.")
	}
	token := os.Getenv("TEL_BOT_TOKEN")
	if token == "" {
		log.Fatal("FATAL: TEL_BOT_TOKEN environment variable not set.")
	}

	// Initialize database connection using the new platform package
	dbConnection, err := database.ConnectDB() // CORRECTED CALL
	if err != nil {
		log.Fatalf("FATAL: Could not initialize database connection: %v", err)
	}
	log.Println("Database connection successful.")

	// Initialize application dependencies using the DI container (wire)
	// This creates all our handlers and use cases.
	botApp, err := di.InitializeBotApp(dbConnection)
	if err != nil {
		log.Fatalf("FATAL: Could not initialize bot dependencies: %v", err)
	}
	log.Println("Application services and handlers initialized successfully.")

	// We need the RegisterUserHandler specifically for the middleware, so we initialize it here as well.
	// In a more advanced setup, the DI container could provide the whole configured middleware chain.
	registerUserHandler := di.InitializeRegisterUserHandler(dbConnection)

	// Initialize the bot instance, passing in the configured handlers
	botInstance, err := telegram.InitializeBot(
		token,
		botApp.CommandHandler,
		botApp.MessageHandler,
		botApp.CallbackHandler,
		registerUserHandler,
	)

	if err != nil {
		log.Fatalf("FATAL: Could not initialize bot: %v", err)
	}

	log.Println("Bot starting...")
	botInstance.Start()
}
