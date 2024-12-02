package tg

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"logs-aggregator-bot/constants"
	"logs-aggregator-bot/provider"
	"logs-aggregator-bot/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type TgHandler struct {
	updates  <-chan tgbotapi.Update
	bot      *tgbotapi.BotAPI
	handler  *services.ApiHandler
	provider *provider.JsonStorageProvider
}

func NewTgHandler(bot *tgbotapi.BotAPI, handler *services.ApiHandler, provider *provider.JsonStorageProvider) *TgHandler {
	updates, _ := bot.GetUpdatesChan(tgbotapi.NewUpdate(0))
	return &TgHandler{bot: bot, handler: handler, provider: provider, updates: updates}
}

func (t *TgHandler) Start(ctx context.Context) {
	fmt.Println("Start work")
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-t.updates:
			if update.CallbackQuery != nil {
				go t.processCallback(update)
				continue
			}
			if update.Message != nil {
				if update.Message.IsCommand() {
					go t.processCommand(update)
					continue
				}
				go t.processMessage(update)
			}
		}
	}
}

func (t *TgHandler) processCallback(update tgbotapi.Update) {
	settings, err := t.provider.GetUserSettings()

	if err != nil {
		logrus.Errorf("Failed to get user settings: %s", err.Error())
		return
	}

	if settings.UserId != update.CallbackQuery.Message.Chat.ID {
		logrus.Errorf("Not granted user callback, skip")
		return
	}

	switch settings.CurrentState {
	case constants.UserStateSelectLogType:
		t.handler.HandleCallbackSelectLogType(update.CallbackQuery.Data)
	case constants.UserStateSelectNewLogDate:
		t.handler.HandleCallbackSelectNewLogDate(update.CallbackQuery.Data)
	case constants.UserStateSelectOldLogDate:
		t.handler.HandleCallbackSelectOldLogDate(update.CallbackQuery.Data)
	default:
		return
	}

	_, err = t.bot.AnswerCallbackQuery(tgbotapi.NewCallback(update.CallbackQuery.ID, "Запрос обработан успешно"))

	if err != nil {
		logrus.Errorf("Failed asnwer callback: %v", err)
	}
}

func (t *TgHandler) processMessage(update tgbotapi.Update) {
	settings, err := t.provider.GetUserSettings()

	if err != nil {
		logrus.Errorf("Failed to get user settings: %s", err.Error())
		return
	}

	if settings.UserId != update.Message.Chat.ID {
		logrus.Errorf("Not granted user callback, skip")
		return
	}

	if settings.CurrentState == constants.UserStateSelectNewLogMessage {
		t.handler.HandleSelectNewLogMessage(update.Message.Text)
	}
}

func (t *TgHandler) processCommand(update tgbotapi.Update) {
	settings, err := t.provider.GetUserSettings()

	if err != nil {
		logrus.Errorf("Failed to get user settings: %s", err.Error())
		return
	}

	if settings.UserId != update.Message.Chat.ID {
		logrus.Errorf("Not granted user callback, skip")
		return
	}

	if update.Message.Command() == string(constants.StartWorkDayCommand) {
		t.handler.HandleStartWorkDayCommand()
	}

	if update.Message.Command() == string(constants.EndWorkDayCommand) {
		t.handler.HandleStopWorkDayCommand()
	}

	if update.Message.Command() == string(constants.GetLogsCommand) {
		t.handler.HandleGetLogsCommand()
	}
}
