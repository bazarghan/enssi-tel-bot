package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/2000ostd/enssi-tel-bot/internal/platform/config" // Import new config package
	"github.com/2000ostd/enssi-tel-bot/internal/platform/database"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/di"
	"github.com/robfig/cron/v3"
	"gopkg.in/telebot.v4"
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

	// 2. Worker needs DB connection and a Telebot instance for the Notifier
	dbConnection, err := database.ConnectDB(cfg.Database)
	if err != nil {
		log.Fatalf("FATAL: Could not initialize database connection: %v", err)
	}

	// 3. Create bot instance for notifier using the token from the config struct
	botInstance, err := telebot.NewBot(telebot.Settings{Token: cfg.Telegram.Token})
	if err != nil {
		log.Fatalf("FATAL: Could not create bot instance for worker: %v", err)
	}

	// Initialize dependencies via DI
	workerApp, err := di.InitializeWorkerApp(dbConnection, botInstance)
	if err != nil {
		log.Fatalf("FATAL: Could not initialize worker dependencies: %v", err)
	}
	log.Println("Worker dependencies initialized.")

	// Schedule the daily review job
	c := cron.New()
	// This job runs every minute for testing. For production, you'd change it to once a day, e.g., "0 7 * * *".
	_, err = c.AddJob("* * * * *", workerApp.TriggerDailyReviewsJob)
	if err != nil {
		log.Fatalf("Could not add daily review job to cron: %v", err)
	}

	c.Start()
	log.Printf("Cron worker started. Daily review job scheduled.")

	// Wait for termination signal to gracefully shut down
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down cron worker...")
	ctx := c.Stop() // Stop the scheduler, waiting for running jobs to complete
	<-ctx.Done()
	log.Println("Worker stopped.")
}

