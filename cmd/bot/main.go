// File: cmd/bot/main.go

package main

import (
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/config"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/database"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/di"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	"log"
	"os"
)

func main() {
	// 1. Load centralized configuration
	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		log.Fatalf("FATAL: Could not load configuration: %v", err)
	}

	// 2. Initialize database connection
	dbConnection, err := database.ConnectDB(cfg.Database)
	if err != nil {
		// Use standard log before our logger is available
		log.Fatalf("FATAL: Could not initialize database connection: %v", err)
	}
	// 2.5 Initialize Logger
	appLogger, err := logger.New(cfg.Log.Level)
	if err != nil {
		log.Fatalf("FATAL: Could not initialize logger: %v", err)
	}

	// --- DI happens here ---
	// We now pass the logger into the dependency injector.
	botApp, err := di.InitializeBotApp(cfg, dbConnection, appLogger) // Pass logger in
	if err != nil {
		log.Fatalf("FATAL: Could not initialize bot dependencies: %v", err)
	}

	appLogger.Info("Logger and dependencies initialized by DI.", "level", cfg.Log.Level)

	// Validate essential config
	if cfg.Telegram.Token == "" {
		appLogger.Error("FATAL: Telegram token (ENSSI_TELEGRAM_TOKEN) is not set.")
		os.Exit(1)
	}

	appLogger.Info("Database connection successful.")

	// 3. Initialize the bot instance
	botInstance, err := telegram.InitializeBot(
		cfg,
		appLogger,
		cfg.Telegram.Token, // <-- ADD THIS ARGUMENT
		botApp.CommandHandler,
		botApp.MessageHandler,
		botApp.CallbackHandler,
		botApp.RegisterUserHandler,
	)
	if err != nil {
		appLogger.Error("FATAL: Could not initialize bot", "error", err)
		os.Exit(1)
	}

	// Initialize the broadcast handler
	broadcastHandler, err := di.InitializeBroadcastHandler(cfg, dbConnection, botInstance, appLogger) // <-- FIX IS HERE
	if err != nil {
		appLogger.Error("FATAL: Could not initialize broadcast handler", "error", err)
		os.Exit(1)
	}
	botApp.MessageHandler.Broadcast = broadcastHandler
	appLogger.Info("Broadcast handler injected successfully.")

	appLogger.Info("Bot starting...")
	botInstance.Start()
}
