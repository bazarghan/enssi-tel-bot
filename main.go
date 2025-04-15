package main

import (
	"log"

	"github.com/2000ostd/enssi-tel-bot/internal/bot"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/handlers"
	"github.com/2000ostd/enssi-tel-bot/internal/db"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbMain, err := db.ConnectDB()
	if err != nil {
		log.Fatalf("Could not initialize database connection: %v", err)
	}
	log.Println("Database connection successful")
	log.Println(dbMain)

	bot, err := bot.InitializeBot()
	if err != nil {
		log.Fatalf("Could not initialize bot: %v", err)
	}
	log.Println("Bot initialization successful")

	handlers.RegisterHandlers(bot)

	bot.Start()
}
