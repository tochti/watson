package watson

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"

	"gopkg.in/telegram-bot-api.v4"
)

type (
	TelegramWebhookHandler func(update tgbotapi.Update)

	TelegramConfig struct {
		Token          string `json:"token"`
		ChatID         int64  `json:"chat_id"`
		WebhookURLHost string `json:"webhook_url_host"`
		WebhookURLPath string `json:"webhook_url_path"`
		WebhookCert    string `json:"webhook_cert"`
		WebhookKey     string `json:"webhook_key"`
	}

	PgSQLConfig struct {
		User     string `json:"user"`
		Password string `json:"password"`
		Host     string `json:"host"`
		Database string `json:"database"`
	}

	FeedsConfig struct {
		Interval string `json:"interval"`
	}

	Config struct {
		Debug    bool           `json:"debug"`
		Telegram TelegramConfig `json:"telegram"`
		PgSQL    PgSQLConfig    `json:"pg_sql"`
		Feeds    FeedsConfig    `json:"feeds"`
	}
)

func ListenTelegramWebhook(config TelegramConfig, telegramClient *tgbotapi.BotAPI, handlers []TelegramWebhookHandler) {

	url := config.WebhookURLHost + config.WebhookURLPath
	_, err := telegramClient.SetWebhook(tgbotapi.NewWebhookWithCert(url, config.WebhookCert))
	if err != nil {
		log.Fatal(err)
	}

	updates := telegramClient.ListenForWebhook(config.WebhookURLPath)
	go http.ListenAndServeTLS(config.WebhookURLHost, config.WebhookCert, config.WebhookKey, nil)

	go func() {
		for update := range updates {
			for _, h := range handlers {
				// Jeder Handler muss sich selbst entscheiden ob er versucht das Update nochmals
				// zu verarbeiten oder nicht daher werden hier keine Fehler abgefragt oder
				// irgendeine Fehlerbehandlung vorgenommen.
				go func() {
					h(update)
				}()
			}

		}
	}()
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

	return u.String()
}
