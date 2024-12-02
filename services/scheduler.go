package services

import (
	"context"
	"fmt"
	"logs-aggregator-bot/constants"
	"logs-aggregator-bot/models"
	"logs-aggregator-bot/provider"
	"logs-aggregator-bot/utils"
	"time"
)

type tgClient interface {
	SendMessage(req *models.SendNotificationRequest) error
}

type SchedulerService struct {
	provider *provider.JsonStorageProvider
	tgClient tgClient
}

func NewSchedulerService(provider *provider.JsonStorageProvider, tgCli tgClient) *SchedulerService {
	return &SchedulerService{provider, tgCli}
}

func (s *SchedulerService) Start(ctx context.Context, doneChan <-chan struct{}) {
	settings, err := s.provider.GetUserSettings()

	if err != nil {
		return
	}

	if settings.WorkStarted.Day() == time.Now().Day() {
		return
	}

	settings.WorkStarted = time.Now()
	err = s.provider.SetUserSettings(settings)

	if err != nil {
		return
	}

	notificationTicker := time.NewTicker(time.Hour)
	for {
		select {
		case <-ctx.Done():
			notificationTicker.Stop()
			return
		case <-doneChan:
			notificationTicker.Stop()
			return
		case <-notificationTicker.C:
			settings, err := s.provider.GetUserSettings()

			if err != nil {
				continue
			}

			settings.CurrentState = constants.UserStateSelectLogType
			settings.NeedWorkLogTo = time.Now()

			err = s.provider.SetUserSettings(settings)

			if err != nil {
				continue
			}

			logs, err := s.provider.GetLogRecords(settings.WorkStarted)

			if err != nil {
				fmt.Println(err.Error())
				continue
			}

			if len(logs) == 0 {
				settings.CurrentState = constants.UserStateSelectNewLogMessage

				_ = s.provider.SetUserSettings(settings)
				err = s.tgClient.SendMessage(&models.SendNotificationRequest{
					ChatId: settings.UserId,
					Body:   fmt.Sprintf("Залогайте вашу работу за период: %s-%s", utils.GetOnlyTime(utils.RoundTimeToMinutes(settings.WorkStarted)), utils.GetOnlyTime(utils.RoundTimeToMinutes(settings.NeedWorkLogTo))),
				})

				if err != nil {
					continue
				}

				continue
			}

			lastLog := 0

			for i, v := range logs {
				if v.EndWorkTime.Compare(logs[lastLog].EndWorkTime) == 1 {
					lastLog = i
				}
			}

			err = s.tgClient.SendMessage(&models.SendNotificationRequest{
				ChatId: settings.UserId,
				Body:   fmt.Sprintf("Ваш последний ворк-лог по работе: %s, время начала: %s,  желаете ли вы продолжить его по времени?", logs[lastLog].Message, utils.RoundTimeToMinutes(logs[lastLog].StartWorkTime)),
				Markup: []models.MarkupData{
					{
						Key:   "Да",
						Value: constants.CallbackParamContinueOldLog,
					},
					{
						Key:   "Нет",
						Value: constants.CallbackParamCreateNewLog,
					},
				},
			})

			if err != nil {
				continue
			}

		}
	}
}
