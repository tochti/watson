package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/tochti/watson/feeds"
	"github.com/uber-go/zap"
	"gopkg.in/telegram-bot-api.v4"

	_ "github.com/lib/pq"
)

type (
	TelegramConfig struct {
		Token  string `json:"token"`
		ChatID int64  `json:"chat_id"`
	}

	PgSQLConfig struct {
		User     string `json:"user"`
		Password string `json:"password"`
		Host     string `json:"host"`
		Database string `json:"database"`
	}

	Config struct {
		Debug    bool           `json:"debug"`
		Telegram TelegramConfig `json:"telegram"`
		PgSQL    PgSQLConfig    `json:"pg_sql"`
	}
)

func main() {
	configFile := flag.String("config", "./watson.json", "Path to config")

	config := ReadConfig(*configFile)

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

	url := PgSQLURL(config.PgSQL)
	db, err := sql.Open("postgres", url)
	if err != nil {
		log.Error("Cannot connect to postgres", zap.Error(err))
		os.Exit(1)
	}

	done := make(chan struct{})
	feeds.StartCtrl(log, db, config.Telegram.ChatID, telegramClient)
	<-done
}

func ReadConfig(p string) Config {
	fh, err := os.Open(p)
	if err != nil {
		log.Fatalf("Cannot open configfile - %v", err)
	}

	cfg := Config{}
	err = json.NewDecoder(fh).Decode(&cfg)

	return cfg
}

func PgSQLURL(c PgSQLConfig) string {
	v := url.Values{}
	v.Set("sslmode", "disable")

	u := url.URL{
		Scheme:     "postgres",
		Host:       c.Host,
		User:       url.UserPassword(c.User, c.Password),
		Path:       "/" + c.Database,
		ForceQuery: true,
		RawQuery:   v.Encode(),
	}

	fmt.Println(u.String())
	return u.String()
}
