package services

import (
	"context"
	"fmt"
	"logs-aggregator-bot/constants"
	"logs-aggregator-bot/models"
	"logs-aggregator-bot/provider"
	"logs-aggregator-bot/utils"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type ApiHandler struct {
	provider      *provider.JsonStorageProvider
	tgClient      tgClient
	scheduler     *SchedulerService
	doneChan      chan<- struct{}
	cachedMessage string
}

func NewApiHandler(provider *provider.JsonStorageProvider, tgClient tgClient, scheduler *SchedulerService) *ApiHandler {
	return &ApiHandler{provider: provider, tgClient: tgClient, scheduler: scheduler, cachedMessage: ""}
}

func (a *ApiHandler) HandleStartWorkDayCommand() {
	fmt.Println("Handle start command")
	settings, err := a.provider.GetUserSettings()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if settings.WorkStarted.Day() == time.Now().Day() {
		fmt.Println("Today you used this")
		_ = a.tgClient.SendMessage(&models.SendNotificationRequest{
			ChatId: settings.UserId,
			Body:   "Вы уже начали свой рабочий день",
		})
		return
	}
	fmt.Println("Prepare scheduler")

	if a.doneChan != nil {
		close(a.doneChan)
	}

	doneChan := make(chan struct{})
	a.doneChan = doneChan

	go a.scheduler.Start(context.TODO(), doneChan)
}

func (a *ApiHandler) HandleStopWorkDayCommand() {
	a.doneChan <- struct{}{}
}

func (a *ApiHandler) HandleCallbackSelectLogType(data string) {
	settings, err := a.provider.GetUserSettings()

	if err != nil {
		return
	}

	if data == constants.CallbackParamContinueOldLog {
		settings.CurrentState = constants.UserStateSelectOldLogDate

		err = a.provider.SetUserSettings(settings)

		if err != nil {
			return
		}

		logs, err := a.provider.GetLogRecords(settings.WorkStarted)

		if err != nil {
			return
		}

		oldLogIndex := 0

		for i, v := range logs {
			if v.EndWorkTime.Compare(logs[oldLogIndex].EndWorkTime) == 1 {
				oldLogIndex = i
			}
		}

		dateIntervals := utils.GetInterval(utils.RoundTimeToMinutes(logs[oldLogIndex].EndWorkTime), settings.NeedWorkLogTo.Add(time.Minute*14), time.Minute*10)
		markup := []models.MarkupData{}

		for _, v := range dateIntervals {
			markup = append(markup, models.MarkupData{
				Key:   utils.GetOnlyTime(v),
				Value: fmt.Sprint(v.UnixMilli()),
			})
		}

		markup = append(markup, models.MarkupData{
			Key:   utils.GetOnlyTime(settings.NeedWorkLogTo),
			Value: fmt.Sprint(settings.NeedWorkLogTo.UnixMilli()),
		})

		err = a.tgClient.SendMessage(&models.SendNotificationRequest{
			ChatId: settings.UserId,
			Body:   fmt.Sprintf("Выберите время, по которую вы продолжали задачу %s", logs[oldLogIndex].Message),
			Markup: markup,
		})

		if err != nil {
			return
		}
	}

	if data == constants.CallbackParamCreateNewLog {
		settings.CurrentState = constants.UserStateSelectNewLogMessage

		err = a.provider.SetUserSettings(settings)

		if err != nil {
			return
		}

		err = a.tgClient.SendMessage(&models.SendNotificationRequest{
			ChatId: settings.UserId,
			Body:   "Введите комментарий к работе, что вы выполняли",
		})

		if err != nil {
			return
		}
	}
}

func (a *ApiHandler) HandleCallbackSelectOldLogDate(data string) {
	settings, err := a.provider.GetUserSettings()

	if err != nil {
		return
	}

	logs, err := a.provider.GetLogRecords(settings.WorkStarted)

	if err != nil {
		return
	}

	oldLogIndex := 0

	for i, v := range logs {
		if v.EndWorkTime.Compare(logs[oldLogIndex].EndWorkTime) == 1 {
			oldLogIndex = i
		}
	}
	parsedLong, err := strconv.ParseInt(data, 10, 64)

	if err != nil {
		return
	}

	parsedTime := time.UnixMilli(parsedLong)

	log := logs[oldLogIndex]
	log.EndWorkTime = parsedTime
	a.provider.UpdateLogRecord(&log)

	if settings.NeedWorkLogTo.Round(time.Second).Compare(parsedTime.Round(time.Second)) > 0 {
		settings.CurrentState = constants.UserStateSelectNewLogMessage

		err = a.provider.SetUserSettings(settings)

		if err != nil {
			return
		}

		err = a.tgClient.SendMessage(&models.SendNotificationRequest{
			ChatId: settings.UserId,
			Body:   fmt.Sprintf("У вас не полностью забит ворклог. Напишите коментарий, что вы делали в промежутке %s-%s", utils.GetOnlyTime(parsedTime), utils.GetOnlyTime(settings.NeedWorkLogTo)),
		})

		if err != nil {
			return
		}
	} else {
		settings.CurrentState = constants.UserStateNone
		err = a.provider.SetUserSettings(settings)
	}

}

func (a *ApiHandler) HandleSelectNewLogMessage(message string) {
	settings, err := a.provider.GetUserSettings()

	if err != nil {
		return
	}

	settings.CurrentState = constants.UserStateSelectNewLogDate

	err = a.provider.SetUserSettings(settings)

	if err != nil {
		return
	}

	logs, err := a.provider.GetLogRecords(settings.WorkStarted)

	if err != nil {
		return
	}

	oldLogIndex := 0

	for i, v := range logs {
		if v.EndWorkTime.Compare(logs[oldLogIndex].EndWorkTime) == 1 {
			oldLogIndex = i
		}
	}

	a.cachedMessage = message

	intervals := []time.Time{}

	if len(logs) > 0 {
		intervals = utils.GetInterval(logs[oldLogIndex].EndWorkTime, settings.NeedWorkLogTo, time.Minute*10)
	} else {
		intervals = utils.GetInterval(settings.WorkStarted, settings.NeedWorkLogTo, time.Minute*10)
	}

	markup := []models.MarkupData{}

	for _, v := range intervals {
		markup = append(markup, models.MarkupData{
			Key:   utils.GetOnlyTime(v),
			Value: fmt.Sprint(v.UnixMilli()),
		})
	}

	markup = append(markup, models.MarkupData{
		Key:   utils.GetOnlyTime(settings.NeedWorkLogTo),
		Value: fmt.Sprint(settings.NeedWorkLogTo.UnixMilli()),
	})

	err = a.tgClient.SendMessage(&models.SendNotificationRequest{
		ChatId: settings.UserId,
		Body:   fmt.Sprintf("Выберите время, по которую вы продолжали делать задачу %s", message),
		Markup: markup,
	})

	if err != nil {
		return
	}
}

func (a *ApiHandler) HandleCallbackSelectNewLogDate(data string) {
	settings, err := a.provider.GetUserSettings()

	if err != nil {
		return
	}

	logs, err := a.provider.GetLogRecords(settings.WorkStarted)

	if err != nil {
		return
	}

	oldLogIndex := 0

	for i, v := range logs {
		if v.EndWorkTime.Compare(logs[oldLogIndex].EndWorkTime) == 1 {
			oldLogIndex = i
		}
	}

	parsedLong, err := strconv.ParseInt(data, 10, 64)

	if err != nil {
		return
	}

	startTime := time.Time{}

	if len(logs) == 0 {
		startTime = settings.WorkStarted
	} else {
		startTime = logs[oldLogIndex].EndWorkTime
	}

	parsedTime := time.UnixMilli(parsedLong)
	newLog := &models.LogsInfoDto{
		Id:            uuid.NewString(),
		StartWorkTime: startTime,
		EndWorkTime:   parsedTime,
		Message:       a.cachedMessage,
	}
	_ = a.provider.InsertNewLogRecord(parsedTime, newLog)
	a.cachedMessage = ""

	if settings.NeedWorkLogTo.Round(time.Second).Compare(parsedTime.Round(time.Second)) <= 0 {
		settings.CurrentState = constants.UserStateNone
		_ = a.provider.SetUserSettings(settings)
		return
	}

	settings.CurrentState = constants.UserStateSelectNewLogMessage
	_ = a.provider.SetUserSettings(settings)

	err = a.tgClient.SendMessage(&models.SendNotificationRequest{
		ChatId: settings.UserId,
		Body:   fmt.Sprintf("У вас не полностью забит ворклог. Напишите коментарий, что вы делали в промежутке %s-%s", utils.GetOnlyTime(parsedTime), utils.GetOnlyTime(settings.NeedWorkLogTo)),
	})

	if err != nil {
		return
	}
}

func (a *ApiHandler) HandleGetLogsCommand() {
	settings, err := a.provider.GetUserSettings()

	if err != nil {
		return
	}

	logs, err := a.provider.GetLogRecords(time.Now())

	if err != nil {
		return
	}

	if len(logs) == 0 {
		err = a.tgClient.SendMessage(&models.SendNotificationRequest{
			ChatId: settings.UserId,
			Body:   "Логов нет",
		})
		return
	}

	firstLog := 0
	lastLog := 0

	for i, v := range logs {
		if v.StartWorkTime.Compare(logs[firstLog].StartWorkTime) == -1 {
			firstLog = i
		}
		if v.EndWorkTime.Compare(logs[lastLog].EndWorkTime) == 1 {
			lastLog = i
		}
	}

	messageText := fmt.Sprintf("Отчет по времени за период: %s-%s:\n", utils.GetOnlyTime(logs[firstLog].StartWorkTime), utils.GetOnlyTime(logs[lastLog].EndWorkTime))

	for _, v := range logs {
		delta := v.EndWorkTime.Sub(v.StartWorkTime)
		diffStr := ""

		if int(delta.Hours()) > 0 {
			diffStr += fmt.Sprintf("%dh", int(delta.Hours()))
		}

		if int(delta.Minutes()) > 0 {
			diffStr += fmt.Sprintf(" %dm", int(delta.Minutes()))
		}

		messageText += fmt.Sprintf("Задача: %s, Начало работ: %s, Конец работ: %s, Затрачено времени: %s \n", v.Message, v.StartWorkTime, v.EndWorkTime, diffStr)
	}

	err = a.tgClient.SendMessage(&models.SendNotificationRequest{
		ChatId: settings.UserId,
		Body:   messageText,
	})

	if err != nil {
		return
	}
}
