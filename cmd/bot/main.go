package main

import (
	"log"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/config"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/database"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/di"
)

func main() {
	// 1. Load centralized configuration
	cfg, err := config.LoadConfig("./configs") // Point to the configs directory
	if err != nil {
		log.Fatalf("FATAL: Could not load configuration: %v", err)
	}

	// Validate essential config
	if cfg.Telegram.Token == "" {
		log.Fatal("FATAL: Telegram token (ENSSI_TELEGRAM_TOKEN) is not set.")
	}

	// 2. Initialize database connection using the config struct
	dbConnection, err := database.ConnectDB(cfg.Database)
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

	// We need the RegisterUserHandler specifically for the middleware
	registerUserHandler := di.InitializeRegisterUserHandler(dbConnection)

	// 3. Initialize the bot instance, passing the token from our config struct
	botInstance, err := telegram.InitializeBot(
		cfg.Telegram.Token,
		botApp.CommandHandler,
		botApp.MessageHandler,
		botApp.CallbackHandler,
		registerUserHandler,
	)
	if err != nil {
		log.Fatalf("FATAL: Could not initialize bot: %v", err)
	}

	// Initialize the broadcast handler
	broadcastHandler, err := di.InitializeBroadcastHandler(dbConnection, botInstance)
	if err != nil {
		log.Fatalf("FATAL: Could not initialize broadcast handler: %v", err)
	}

	// Manually inject the broadcast handler into our message handler
	botApp.MessageHandler.Broadcast = broadcastHandler
	log.Println("Broadcast handler injected successfully.")

	log.Println("Bot starting...")
	botInstance.Start()
}
