package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/2000ostd/enssi-tel-bot/internal/platform/database"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/di"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"gopkg.in/telebot.v4"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found.")
	}

	// Worker needs DB connection and a Telebot instance for the Notifier
	dbConnection, err := database.ConnectDB()
	if err != nil {
		log.Fatalf("FATAL: Could not initialize database connection: %v", err)
	}

	token := os.Getenv("TEL_BOT_TOKEN")
	if token == "" {
		log.Fatal("FATAL: TEL_BOT_TOKEN environment variable not set.")
	}
	botInstance, err := telebot.NewBot(telebot.Settings{Token: token})
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
	_, err = c.AddJob("* * * * *", workerApp.TriggerDailyReviewsJob) // Every day at 7 AM UTC
	if err != nil {
		log.Fatalf("Could not add daily review job to cron: %v", err)
	}

	c.Start()
	log.Printf("Cron worker started. Daily review job scheduled for 07:00 UTC.")

	// Wait for termination signal to gracefully shut down
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down cron worker...")
	ctx := c.Stop() // Stop the scheduler, waiting for running jobs to complete
	<-ctx.Done()
	log.Println("Worker stopped.")
}
