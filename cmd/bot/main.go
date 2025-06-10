package main

import (
	"github.com/2000ostd/enssi-tel-bot/internal/platform/config"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	"log"
)

func main() {
	log.Println("Starting bot application...")

	// 1. Initialize Logger
	appLogger := logger.New()
	appLogger.Info("Logger initialized")

	// 2. Load Configuration
	cfg, err := config.Load("./configs") // Path to config directory
	if err != nil {
		appLogger.Error("Failed to load configuration", "error", err)
		return
	}
	appLogger.Info("Configuration loaded successfully")

	// This is during Test and its Redundant code
	if cfg != nil {
		appLogger.Info("This is for the phase 0 just for The test")
	}

	// Later, we will replace this with a call to a dependency injection container
	// db, err := database.NewConnection(cfg.Postgres)
	// if err != nil { ... }
	//
	// bot, err := di.InitializeBot(cfg, appLogger, db)
	// if err != nil { ... }
	//
	// bot.Start()

	appLogger.Info("Scaffolding complete. Application setup is runnable.")
}
