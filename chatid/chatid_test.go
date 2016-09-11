package chatid

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/tochti/watson"
	"github.com/uber-go/zap"
	"gopkg.in/telegram-bot-api.v4"
)

func TestChatID(t *testing.t) {
	log := testLogger()
	config := readTestConfig()
	bot := tgbotapi.NewBotAPI(config.Telegram.Token)
	watson.ListenTelegramWebhook(config, bot, Handler(log, bot))

	msg := tgbotapi.NewMessage(42, "chatid?")
	sendMessage(config, msg)
}

func sendMessage(cfg watson.Config, msg *tgbotapi.Message) {
	// Load CA cert
	caCert, err := ioutil.ReadFile(cfg.Telegram)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	// Do GET something
	url := cfg.Telegram.WebhookURLHost + cfg.Telegram.WebhookURLPath
	resp, err := client.POST(url, "application/json", body)
	if err != nil {
		log.Fatal(err)
	}
}

func readTestConfig() watson.Config {
	path := os.Getenv("TEST_CONFIG_PATH")
	return watson.ReadConfig(path)
}

func testLogger() zap.Logger {
	return zap.New(
		zap.NewJSONEncoder(zap.NoTime),
	)
}
