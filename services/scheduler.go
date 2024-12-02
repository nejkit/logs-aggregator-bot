package services

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
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
		logrus.Errorf("Failed to get user settings: %v", err)
		return
	}

	if settings.WorkStarted.Day() == time.Now().Day() {
		logrus.Warn("Scheduler is already running, skip request")
		return
	}

	settings.WorkStarted = time.Now()
	err = s.provider.SetUserSettings(settings)

	if err != nil {
		logrus.Errorf("Failed to set user settings: %v", err)
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
				logrus.Errorf("Failed to get user settings: %v", err)
				continue
			}

			settings.CurrentState = constants.UserStateSelectLogType
			settings.NeedWorkLogTo = time.Now()

			err = s.provider.SetUserSettings(settings)

			if err != nil {
				logrus.Errorf("Failed to set user settings: %v", err)
				continue
			}

			logs, err := s.provider.GetLogRecords(settings.WorkStarted)

			if err != nil {
				logrus.Errorf("Failed to get logs: %v", err)
				continue
			}

			if len(logs) == 0 {
				settings.CurrentState = constants.UserStateSelectNewLogMessage

				err = s.provider.SetUserSettings(settings)

				if err != nil {
					logrus.Errorf("Failed to set user settings: %v", err)
					continue
				}

				err = s.tgClient.SendMessage(&models.SendNotificationRequest{
					ChatId: settings.UserId,
					Body:   fmt.Sprintf("Залогайте вашу работу за период: %s-%s", utils.GetOnlyTime(settings.WorkStarted), utils.GetOnlyTime(settings.NeedWorkLogTo)),
				})

				if err != nil {
					logrus.Errorf("Failed to send message to user: %v", err)
					continue
				}

				continue
			}

			lastLog := getLastLog(logs)

			err = s.tgClient.SendMessage(&models.SendNotificationRequest{
				ChatId: settings.UserId,
				Body:   fmt.Sprintf("Ваш последний ворк-лог по работе: %s, время начала: %s,  желаете ли вы продолжить его по времени?", lastLog.Message, utils.GetOnlyTime(lastLog.StartWorkTime)),
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
				logrus.Errorf("Failed to send message to user: %v", err)
				continue
			}

		}
	}
}
