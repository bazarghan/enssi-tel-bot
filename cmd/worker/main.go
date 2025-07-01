// File: cmd/worker/main.go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/2000ostd/enssi-tel-bot/internal/platform/config"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/database"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/di"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	"github.com/robfig/cron/v3"
	"gopkg.in/telebot.v4"
)

func main() {
	// 1. Load centralized configuration
	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		log.Fatalf("FATAL: Could not load configuration: %v", err)
	}

	// Validate essential config
	if cfg.Telegram.Token == "" {
		log.Fatal("FATAL: Telegram token (ENSSI_TELEGRAM_TOKEN) is not set.")
	}

	// 2. Initialize database connection
	dbConnection, err := database.ConnectDB(cfg.Database)
	if err != nil {
		log.Fatalf("FATAL: Could not initialize database connection: %v", err)
	}

	// 3. Create bot instance for notifier
	botInstance, err := telebot.NewBot(telebot.Settings{Token: cfg.Telegram.Token})
	if err != nil {
		log.Fatalf("FATAL: Could not create bot instance for worker: %v", err)
	}

	// --- DI happens here ---
	appLogger := logger.New(logger.LevelInfo, os.Stdout) // Create logger instance
	if cfg.Log.Level == "debug" {
		appLogger = logger.New(logger.LevelDebug, os.Stdout)
	}

	// --- DI happens here ---
	workerApp, err := di.InitializeWorkerApp(cfg, dbConnection, botInstance, appLogger) // Pass logger in
	if err != nil {
		log.Fatalf("FATAL: Could not initialize worker dependencies: %v", err)
	}

	appLogger.Info("Worker dependencies initialized.")

	// Schedule the daily review job
	c := cron.New()
	_, err = c.AddJob("0 7 * * *", workerApp.TriggerDailyReviewsJob) // Changed to once a day
	if err != nil {
		appLogger.Error("Could not add daily review job to cron", "error", err)
	}

	c.Start()
	appLogger.Info("Cron worker started. Daily review job scheduled.")

	// Wait for termination signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	appLogger.Info("Shutting down cron worker...")
	ctx := c.Stop()
	<-ctx.Done()
	appLogger.Info("Worker stopped.")
}
