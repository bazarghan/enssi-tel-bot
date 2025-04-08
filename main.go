package main

import (
	"log"
	"os"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/db"
	"github.com/joho/godotenv"
	"gopkg.in/telebot.v4"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := db.ConnectDB()
	if err != nil {
		log.Fatalf("Could not initialize database connection: %v", err)
	}

	// Use the database connection
	log.Println("Database connection available.")
	log.Println(db)

	conf := telebot.Settings{
		Token:  os.Getenv("TEL_BOT_TOKEN"),
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := telebot.NewBot(conf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Bot started!")

	bot.Start()
}
