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
	storageProvider := initStorage()
	token := os.Getenv("API_TOKEN")

	if token == "" {
		panic("not passed API_TOKEN")
	}

	tgBot, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		panic(err)
	}

	tgClient := tg.NewTgClient(tgBot)

	scheduler := services.NewSchedulerService(storageProvider, tgClient)
	handler := services.NewApiHandler(storageProvider, tgClient, scheduler)

	tgHandler := tg.NewTgHandler(tgBot, handler, storageProvider)

	tgHandler.Start(context.TODO())
}

func initStorage() *provider.JsonStorageProvider {
	storageProvider, err := provider.NewJsonStorageProvider()

	if err != nil {
		panic(err)
	}

	superUserId := os.Getenv("SUPER_USER_ID")

	if superUserId == "" {
		panic("not passed SUPER_USER_ID")
	}

	longSuperUser, err := strconv.ParseInt(superUserId, 10, 64)

	if err != nil {
		panic("SUPER_USER_ID is not a number")
	}

	currentSettings, err := storageProvider.GetUserSettings()

	if err != nil {
		panic(err)
	}

	if currentSettings.UserId != longSuperUser {
		err = storageProvider.SetUserSettings(&models.UserSettingsDto{
			UserId:       longSuperUser,
			CurrentState: constants.UserStateNone,
		})

		if err != nil {
			panic(err)
		}
	}

	return storageProvider
}
