package tg

import (
	"logs-aggregator-bot/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type TgClient struct {
	bot *tgbotapi.BotAPI
}

func NewTgClient(bot *tgbotapi.BotAPI) *TgClient {
	return &TgClient{bot: bot}
}

func (t *TgClient) SendMessage(req *models.SendNotificationRequest) error {
	msg := tgbotapi.NewMessage(req.ChatId, req.Body)

	if len(req.Markup) > 0 {
		markup := tgbotapi.NewInlineKeyboardMarkup()

		for _, v := range req.Markup {
			markup.InlineKeyboard = append(markup.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(v.Key, v.Value)))
		}
		msg.ReplyMarkup = markup
	}

	_, err := t.bot.Send(msg)
	return err
}
