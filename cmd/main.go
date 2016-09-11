package main

import (
	"database/sql"
	"flag"
	"os"

	"github.com/tochti/watson"
	"github.com/tochti/watson/chatid"
	"github.com/tochti/watson/feeds"
	"github.com/uber-go/zap"
	"gopkg.in/telegram-bot-api.v4"

	_ "github.com/lib/pq"
)

func main() {
	configFile := flag.String("config", "./watson.json", "Path to config")
	flag.Parse()

	config := watson.ReadConfig(*configFile)

	log := zap.New(
		zap.NewJSONEncoder(),
	)
	if config.Debug {
		log.SetLevel(zap.DebugLevel)
	}

	telegramClient, err := tgbotapi.NewBotAPI(config.Telegram.Token)
	if err != nil {
		log.Error("Cannot start telegram bot", zap.Error(err))
		os.Exit(1)
	}

	url := watson.PgSQLURL(config.PgSQL)
	db, err := sql.Open("postgres", url)
	if err != nil {
		log.Error("Cannot connect to postgres", zap.Error(err))
		os.Exit(1)
	}

	handlers := []watson.TelegramWebhookHandler{
		chatid.Handler(log, telegramClient),
	}

	watson.ListenTelegramWebhook(config.Telegram, telegramClient, handlers)
	feeds.StartCtrl(log, db, telegramClient, config.Telegram.ChatID, config.Feeds.Interval)

	done := make(chan struct{})
	<-done
}
