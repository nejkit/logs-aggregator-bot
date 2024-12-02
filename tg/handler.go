package tg

import (
	"context"
	"fmt"
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
				go t.processCalback(update)
				continue
			}
			if update.Message.IsCommand() {
				go t.processCommand(update)
				continue
			}
			go t.processMessage(update)
		}
	}
}

func (t *TgHandler) processCalback(update tgbotapi.Update) {
	settings, err := t.provider.GetUserSettings()

	if err != nil {
		return
	}

	if settings.UserId != update.CallbackQuery.Message.Chat.ID {
		return
	}

	switch settings.CurrentState {
	case constants.UserStateSelectLogType:
		t.handler.HandleCallbackSelectLogType(update.CallbackQuery.Data)
	case constants.UserStateSelectNewLogDate:
		t.handler.HandleCallbackSelectNewLogDate(update.CallbackQuery.Data)
	case constants.UserStateSelectOldLogDate:
		t.handler.HandleCallbackSelectOldLogDate(update.CallbackQuery.Data)
	}

	t.bot.AnswerCallbackQuery(tgbotapi.NewCallback(update.CallbackQuery.ID, "Process"))
}

func (t *TgHandler) processMessage(update tgbotapi.Update) {
	settings, err := t.provider.GetUserSettings()

	if err != nil {
		return
	}

	if settings.UserId != update.Message.Chat.ID {
		return
	}

	if settings.CurrentState == constants.UserStateSelectNewLogMessage {
		t.handler.HandleSelectNewLogMessage(update.Message.Text)
	}
}

func (t *TgHandler) processCommand(update tgbotapi.Update) {
	if update.Message.Command() == "start_work_day" {
		t.handler.HandleStartWorkDayCommand()
	}

	if update.Message.Command() == "end_work_day" {
		t.handler.HandleStopWorkDayCommand()
	}

	if update.Message.Command() == "get_today_logs" {
		t.handler.HandleGetLogsCommand()
	}
}
