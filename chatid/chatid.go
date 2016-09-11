package chatid

import (
	"fmt"

	"github.com/tochti/watson"
	"github.com/uber-go/zap"

	"gopkg.in/telegram-bot-api.v4"
)

const TRIGGER = "chatid?"

func Handler(log zap.Logger, telegramClient *tgbotapi.BotAPI) watson.TelegramWebhookHandler {
	return func(update tgbotapi.Update) {
		if update.Message == nil {
			return
		}
		msg := update.Message

		if msg.Text != TRIGGER {
			return
		}
		log.Debug("Received chat id request")

		text := fmt.Sprintf("chat id: %v", msg.Chat.ID)
		newMsg := tgbotapi.NewMessage(msg.Chat.ID, text)

		_, err := telegramClient.Send(newMsg)
		if err != nil {
			log.Error("Cannot send chat id", zap.Error(err))
		}
	}

}
