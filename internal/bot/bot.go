package bot

import (
	"os"
	"time"

	"gopkg.in/telebot.v4"
)

func InitializeBot() (*telebot.Bot, error) {
	conf := telebot.Settings{
		Token:  os.Getenv("TEL_BOT_TOKEN"),
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := telebot.NewBot(conf)
	if err != nil {
		return nil, err
	}

	return bot, nil
}
