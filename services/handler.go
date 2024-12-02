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
	"github.com/sirupsen/logrus"
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
	settings, err := a.provider.GetUserSettings()

	if err != nil {
		logrus.Errorf("Failed to get user settings: %v", err)
		return
	}

	if settings.WorkStarted.Day() == time.Now().Day() {
		fmt.Println("Today you used this")
		err = a.tgClient.SendMessage(&models.SendNotificationRequest{
			ChatId: settings.UserId,
			Body:   "Вы уже начали свой рабочий день",
		})

		if err != nil {
			logrus.Errorf("Failed to send message: %v", err)
		}

		return
	}

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

func (a *ApiHandler) HandleDeleteLogsCommand() {
	settings, err := a.provider.GetUserSettings()

	if err != nil {
		logrus.Errorf("Failed to get user settings: %v", err)
		return
	}

	settings.CurrentState = constants.UserStateSelectLogsToDelete

	dates, err := a.provider.GetDatesWithLogs()

	if err != nil {
		logrus.Errorf("Failed to get dates: %v", err)
		return
	}

	if len(dates) == 0 {
		err = a.tgClient.SendMessage(&models.SendNotificationRequest{
			ChatId: settings.UserId,
			Body:   "Нет логов для удаления",
		})

		if err != nil {
			logrus.Errorf("Failed to send message: %v", err)
		}

		return
	}

	err = a.provider.SetUserSettings(settings)

	if err != nil {
		logrus.Errorf("Failed to set user settings: %v", err)
		return
	}

	var markup []models.MarkupData

	for i, date := range dates {
		if i == 5 {
			break
		}

		markup = append(markup, models.MarkupData{
			Key:   date,
			Value: date,
		})
	}

	markup = append(markup, models.MarkupData{
		Key:   "Завершить",
		Value: constants.CallbackStopDeleteLogs,
	})

	err = a.tgClient.SendMessage(&models.SendNotificationRequest{
		ChatId: settings.UserId,
		Body:   "Выберите елементы для их удаления",
		Markup: markup,
	})

	if err != nil {
		logrus.Errorf("Failed to send message: %v", err)
	}
}

func (a *ApiHandler) HandleCallbackSelectLogType(data string) {
	settings, err := a.provider.GetUserSettings()

	if err != nil {
		logrus.Errorf("Failed to get user settings: %v", err)
		return
	}

	switch data {
	case constants.CallbackParamContinueOldLog:
		settings.CurrentState = constants.UserStateSelectOldLogDate

		err = a.provider.SetUserSettings(settings)

		if err != nil {
			logrus.Errorf("Failed to set user settings: %v", err)
			return
		}

		logs, err := a.provider.GetLogRecords(settings.WorkStarted)

		if err != nil {
			logrus.Errorf("Failed to get logs: %v", err)
			return
		}

		lastLog := getLastLog(logs)
		dateIntervals := utils.GetInterval(utils.RoundTimeToMinutes(lastLog.EndWorkTime), settings.NeedWorkLogTo, time.Minute*10)
		var markup []models.MarkupData

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
			Body:   fmt.Sprintf("Выберите время, по которую вы продолжали задачу %s", lastLog.Message),
			Markup: markup,
		})

		if err != nil {
			logrus.Errorf("Failed to send message: %v", err)
			return
		}
	}

	if data == constants.CallbackParamCreateNewLog {
		settings.CurrentState = constants.UserStateSelectNewLogMessage

		err = a.provider.SetUserSettings(settings)

		if err != nil {
			logrus.Errorf("Failed to set user settings: %v", err)
			return
		}

		err = a.tgClient.SendMessage(&models.SendNotificationRequest{
			ChatId: settings.UserId,
			Body:   "Введите комментарий к работе, что вы выполняли",
		})

		if err != nil {
			logrus.Errorf("Failed to send message: %v", err)
			return
		}
	}
}

func (a *ApiHandler) HandleDeleteCallbackParam(data string) bool {
	if data == constants.CallbackStopDeleteLogs {
		settings, err := a.provider.GetUserSettings()

		if err != nil {
			logrus.Errorf("Failed to get user settings: %v", err)
			return false
		}

		settings.CurrentState = constants.UserStateNone
		err = a.provider.SetUserSettings(settings)

		if err != nil {
			logrus.Errorf("Failed to set user settings: %v", err)
			return false
		}
		return false
	}

	err := a.provider.DeleteLogsByDate(data)

	if err != nil {
		logrus.Errorf("failed delete logs: %v", err)
		return false
	}

	return true
}

func (a *ApiHandler) HandleCallbackSelectOldLogDate(data string) {
	settings, err := a.provider.GetUserSettings()

	if err != nil {
		logrus.Errorf("Failed to get user settings: %v", err)
		return
	}

	logs, err := a.provider.GetLogRecords(settings.WorkStarted)

	if err != nil {
		logrus.Errorf("Failed to get logs: %v", err)
		return
	}

	oldLog := getLastLog(logs)
	parsedLong, err := strconv.ParseInt(data, 10, 64)

	if err != nil {
		logrus.Errorf("Failed to parse date: %v", err)
		return
	}

	parsedTime := time.UnixMilli(parsedLong)

	oldLog.EndWorkTime = parsedTime
	err = a.provider.UpdateLogRecord(&oldLog)

	if err != nil {
		logrus.Errorf("Failed to update old log record: %v", err)
		return
	}

	if settings.NeedWorkLogTo.Round(time.Second).Compare(parsedTime.Round(time.Second)) > 0 {
		settings.CurrentState = constants.UserStateSelectNewLogMessage

		err = a.provider.SetUserSettings(settings)

		if err != nil {
			logrus.Errorf("Failed to set user settings: %v", err)
			return
		}

		err = a.tgClient.SendMessage(&models.SendNotificationRequest{
			ChatId: settings.UserId,
			Body:   fmt.Sprintf("У вас не полностью забит ворклог. Напишите коментарий, что вы делали в промежутке %s-%s", utils.GetOnlyTime(parsedTime), utils.GetOnlyTime(settings.NeedWorkLogTo)),
		})

		if err != nil {
			logrus.Errorf("Failed to send message: %v", err)
			return
		}
	} else {
		settings.CurrentState = constants.UserStateNone
		err = a.provider.SetUserSettings(settings)

		if err != nil {
			logrus.Errorf("Failed to set user settings: %v", err)
		}
	}

}

func (a *ApiHandler) HandleSelectNewLogMessage(message string) {
	settings, err := a.provider.GetUserSettings()

	if err != nil {
		logrus.Errorf("failed to get user settings: %v", err)
		return
	}

	settings.CurrentState = constants.UserStateSelectNewLogDate

	err = a.provider.SetUserSettings(settings)

	if err != nil {
		logrus.Errorf("Failed to set user settings: %v", err)
		return
	}

	logs, err := a.provider.GetLogRecords(settings.WorkStarted)

	if err != nil {
		logrus.Errorf("Failed to get logs: %v", err)
		return
	}

	a.cachedMessage = message

	var intervals []time.Time

	if len(logs) > 0 {
		lastLog := getLastLog(logs)
		intervals = utils.GetInterval(lastLog.EndWorkTime, settings.NeedWorkLogTo, time.Minute*10)
	} else {
		intervals = utils.GetInterval(settings.WorkStarted, settings.NeedWorkLogTo, time.Minute*10)
	}

	var markup []models.MarkupData

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
		logrus.Errorf("Failed to send message: %v", err)
		return
	}
}

func (a *ApiHandler) HandleCallbackSelectNewLogDate(data string) {
	settings, err := a.provider.GetUserSettings()

	if err != nil {
		logrus.Errorf("failed to get user settings: %v", err)
		return
	}

	logs, err := a.provider.GetLogRecords(settings.WorkStarted)

	if err != nil {
		logrus.Errorf("Failed to get logs: %v", err)
		return
	}

	parsedLong, err := strconv.ParseInt(data, 10, 64)

	if err != nil {
		return
	}

	startTime := time.Time{}

	if len(logs) == 0 {
		startTime = settings.WorkStarted
	} else {
		startTime = getLastLog(logs).EndWorkTime
	}

	parsedTime := time.UnixMilli(parsedLong)
	newLog := &models.LogsInfoDto{
		Id:            uuid.NewString(),
		StartWorkTime: startTime,
		EndWorkTime:   parsedTime,
		Message:       a.cachedMessage,
	}
	err = a.provider.InsertNewLogRecord(parsedTime, newLog)
	a.cachedMessage = ""

	if err != nil {
		logrus.Errorf("Failed to insert new log record: %v", err)
	}

	if settings.NeedWorkLogTo.Round(time.Second).Compare(parsedTime.Round(time.Second)) <= 0 {
		settings.CurrentState = constants.UserStateNone
		err = a.provider.SetUserSettings(settings)

		if err != nil {
			logrus.Errorf("Failed set user settings: %v", err)
		}

		return
	}

	settings.CurrentState = constants.UserStateSelectNewLogMessage
	err = a.provider.SetUserSettings(settings)

	if err != nil {
		logrus.Errorf("Failed to set user settings: %v", err)
		return
	}

	err = a.tgClient.SendMessage(&models.SendNotificationRequest{
		ChatId: settings.UserId,
		Body:   fmt.Sprintf("У вас не полностью забит ворклог. Напишите коментарий, что вы делали в промежутке %s-%s", utils.GetOnlyTime(parsedTime), utils.GetOnlyTime(settings.NeedWorkLogTo)),
	})

	if err != nil {
		logrus.Errorf("Failed to send message: %v", err)
		return
	}
}

func (a *ApiHandler) HandleGetLogsCommand() {
	settings, err := a.provider.GetUserSettings()

	if err != nil {
		logrus.Errorf("failed to get user settings: %v", err)
		return
	}

	logs, err := a.provider.GetLogRecords(time.Now())

	if err != nil {
		logrus.Errorf("Failed to get logs: %v", err)
		return
	}

	if len(logs) == 0 {
		err = a.tgClient.SendMessage(&models.SendNotificationRequest{
			ChatId: settings.UserId,
			Body:   "Логов нет",
		})
		return
	}

	messageText := constructLogsTable(logs)

	err = a.tgClient.SendMessage(&models.SendNotificationRequest{
		ChatId: settings.UserId,
		Body:   messageText,
	})

	if err != nil {
		logrus.Errorf("Failed to send message: %v", err)
		return
	}
}

func (a *ApiHandler) HandleGetAllLogsCommand() {
	settings, err := a.provider.GetUserSettings()

	if err != nil {
		logrus.Errorf("failed to get user settings: %v", err)
		return
	}

	availableDates, err := a.provider.GetDatesWithLogs()

	if err != nil {
		logrus.Errorf("Failed to get dates: %v", err)
		return
	}

	if len(availableDates) == 0 {
		err = a.tgClient.SendMessage(&models.SendNotificationRequest{
			ChatId: settings.UserId,
			Body:   "У вас нет логов за период использования приложения",
		})

		if err != nil {
			logrus.Errorf("Failed to send message: %v", err)
		}
		return
	}

	var markup []models.MarkupData
	for _, date := range availableDates {
		markup = append(markup, models.MarkupData{
			Key:   date,
			Value: date,
		})
	}

	settings.CurrentState = constants.UserStateSelectLogDate

	err = a.provider.SetUserSettings(settings)

	if err != nil {
		logrus.Errorf("Failed to set user settings: %v", err)
		return
	}

	err = a.tgClient.SendMessage(&models.SendNotificationRequest{
		ChatId: settings.UserId,
		Body:   "Выберите дату, за которую вы хотите получить логи",
		Markup: markup,
	})

	if err != nil {
		logrus.Errorf("Failed to send message: %v", err)
	}
}

func (a *ApiHandler) HandleCallbackWithGetLog(data string) {
	settings, err := a.provider.GetUserSettings()

	if err != nil {
		logrus.Errorf("failed to get user settings: %v", err)
		return
	}

	settings.CurrentState = constants.UserStateNone

	err = a.provider.SetUserSettings(settings)

	if err != nil {
		logrus.Errorf("Failed to set user settings: %v", err)
		return
	}

	parsedDate, err := time.Parse("2006-01-02", data)

	if err != nil {
		logrus.Errorf("Failed to parse date: %v", err)
		return
	}

	logs, err := a.provider.GetLogRecords(parsedDate)

	if err != nil {
		logrus.Errorf("Failed to get logs: %v", err)
		return
	}

	if len(logs) == 0 {
		err = a.tgClient.SendMessage(&models.SendNotificationRequest{
			ChatId: settings.UserId,
			Body:   "Логов за указанный период не найдено. Возможно файл с логами был очищен",
		})

		if err != nil {
			logrus.Errorf("Failed to send message: %v", err)
		}
		return
	}

	messageText := constructLogsTable(logs)

	err = a.tgClient.SendMessage(&models.SendNotificationRequest{
		ChatId: settings.UserId,
		Body:   messageText,
	})

	if err != nil {
		logrus.Errorf("failed to sent message: %v", err)
	}
}

func getFirstLog(logs []models.LogsInfoDto) models.LogsInfoDto {
	firstLogIndex := 0

	for i, v := range logs {
		if v.StartWorkTime.Compare(logs[firstLogIndex].StartWorkTime) == -1 {
			firstLogIndex = i
		}
	}

	return logs[firstLogIndex]
}

func getLastLog(logs []models.LogsInfoDto) models.LogsInfoDto {
	lastLogIndex := 0

	for i, v := range logs {
		if v.EndWorkTime.Compare(logs[lastLogIndex].EndWorkTime) == 1 {
			lastLogIndex = i
		}
	}

	return logs[lastLogIndex]
}

func constructLogsTable(logs []models.LogsInfoDto) string {
	firstLog := getFirstLog(logs)
	lastLog := getLastLog(logs)

	messageText := fmt.Sprintf("Отчет по времени за период: %s-%s:\n", utils.GetOnlyTime(firstLog.StartWorkTime), utils.GetOnlyTime(lastLog.EndWorkTime))

	for _, v := range logs {
		delta := v.EndWorkTime.Sub(v.StartWorkTime)
		diffStr := ""

		if int(delta.Hours()) > 0 {
			diffStr += fmt.Sprintf("%dh", int(delta.Hours()))
		}

		if int(delta.Minutes()) > 0 {
			diffStr += fmt.Sprintf(" %dm", int(delta.Minutes()))
		}

		messageText += fmt.Sprintf("Задача: %s, Начало работ: %s, Конец работ: %s, Затрачено времени: %s \n", v.Message, utils.GetOnlyTime(v.StartWorkTime), utils.GetOnlyTime(v.EndWorkTime), diffStr)
	}

	return messageText
}
