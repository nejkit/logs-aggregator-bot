package main

import (
	"context"
	"logs-aggregator-bot/constants"
	"logs-aggregator-bot/models"
	"logs-aggregator-bot/provider"
	"logs-aggregator-bot/services"
	"logs-aggregator-bot/tg"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func main() {
	provider := provider.NewJsonStorageProvider()

	superUserCreds := os.Getenv("SUPER_USER_ID")

	longSuperUser, err := strconv.ParseInt(superUserCreds, 10, 64)

	if err != nil {
		panic(err)
	}

	currentSettings, err := provider.GetUserSettings()

	if err != nil {
		panic(err)
	}

	if currentSettings.UserId != longSuperUser {
		err = provider.SetUserSettings(&models.UserSettingsDto{
			UserId:       longSuperUser,
			CurrentState: constants.UserStateNone,
		})

		if err != nil {
			panic(err)
		}
	}

	token := os.Getenv("API_TOKEN")

	tgbot, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		panic(err)
	}

	tgClient := tg.NewTgClient(tgbot)

	scheduler := services.NewSchedulerService(provider, tgClient)
	handler := services.NewApiHandler(provider, tgClient, scheduler)

	tgHandler := tg.NewTgHandler(tgbot, handler, provider)

	tgHandler.Start(context.TODO())
}
