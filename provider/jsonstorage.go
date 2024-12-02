package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"logs-aggregator-bot/models"
	"logs-aggregator-bot/utils"
	"os"
	"time"
)

const (
	logFileNavigationFile = "logs_navigation.json"
	logFilePatternFile    = "logs_%s.json"
	userSettingsFile      = "user.json"
)

type JsonStorageProvider struct {
}

func NewJsonStorageProvider() (*JsonStorageProvider, error) {
	content := []byte("{}")
	if _, err := os.Stat(userSettingsFile); errors.Is(err, os.ErrNotExist) {
		_, err = os.Create(userSettingsFile)

		if err != nil {
			return nil, err
		}

		err = os.WriteFile(userSettingsFile, content, 0644)

		if err != nil {
			return nil, err
		}
	}

	if _, err := os.Stat(logFileNavigationFile); errors.Is(err, os.ErrNotExist) {
		_, err = os.Create(logFileNavigationFile)

		if err != nil {
			return nil, err
		}

		err = os.WriteFile(logFileNavigationFile, content, 0644)

		if err != nil {
			return nil, err
		}
	}

	return &JsonStorageProvider{}, nil
}

func (j *JsonStorageProvider) GetUserSettings() (*models.UserSettingsDto, error) {
	content, err := os.ReadFile(userSettingsFile)

	if err != nil {
		return nil, err
	}

	var settings *models.UserSettingsDto

	err = json.Unmarshal(content, &settings)

	if err != nil {
		return nil, err
	}

	return settings, nil
}

func (j *JsonStorageProvider) SetUserSettings(dto *models.UserSettingsDto) error {
	data, err := json.Marshal(dto)

	if err != nil {
		return err
	}

	err = os.WriteFile(userSettingsFile, data, 0644)

	if err != nil {
		return err
	}

	return nil
}

func (j *JsonStorageProvider) InsertNewLogRecord(date time.Time, log *models.LogsInfoDto) error {
	logFile, err := j.getLogFileByDate(date)

	if err != nil {
		return err
	}

	logContent, err := os.ReadFile(logFile)

	if err != nil {
		return err
	}

	var logData []models.LogsInfoDto

	err = json.Unmarshal(logContent, &logData)

	if err != nil {
		return err
	}
	logData = append(logData, *log)

	logContent, err = json.Marshal(logData)

	if err != nil {
		return err
	}

	err = os.WriteFile(logFile, logContent, 0644)

	if err != nil {
		return err
	}

	return nil
}

func (j *JsonStorageProvider) UpdateLogRecord(log *models.LogsInfoDto) error {
	logFile, err := j.getLogFileByDate(log.StartWorkTime)

	if err != nil {
		return err
	}

	logContent, err := os.ReadFile(logFile)

	if err != nil {
		return err
	}

	var logData []*models.LogsInfoDto

	err = json.Unmarshal(logContent, &logData)

	if err != nil {
		return err
	}

	for _, v := range logData {
		if log.Id == v.Id {
			v.EndWorkTime = log.EndWorkTime
		}
	}

	logContent, err = json.Marshal(logData)

	if err != nil {
		return err
	}

	err = os.WriteFile(logFile, logContent, 0644)

	if err != nil {
		return err
	}

	return nil
}

func (j *JsonStorageProvider) GetLogRecords(date time.Time) ([]models.LogsInfoDto, error) {
	logFile, err := j.getLogFileByDate(date)

	if err != nil {
		return nil, err
	}

	logContent, err := os.ReadFile(logFile)

	if err != nil {
		return nil, err
	}

	var logData []models.LogsInfoDto

	err = json.Unmarshal(logContent, &logData)

	if err != nil {
		return nil, err
	}

	return logData, nil
}

func (j *JsonStorageProvider) GetDatesWithLogs() ([]string, error) {
	content, err := os.ReadFile(logFileNavigationFile)

	if err != nil {
		return nil, err
	}

	var navigationDto models.LogsNavigationDto
	err = json.Unmarshal(content, &navigationDto)

	if err != nil {
		return nil, err
	}

	if navigationDto.Date == nil {
		return []string{}, nil
	}

	dates := make([]string, 0)
	for s, _ := range navigationDto.Date {
		dates = append(dates, s)
	}

	return dates, nil
}

func (j *JsonStorageProvider) DeleteLogsByDate(date string) error {
	content, err := os.ReadFile(logFileNavigationFile)

	if err != nil {
		return err
	}

	var navigationDto models.LogsNavigationDto
	err = json.Unmarshal(content, &navigationDto)
	if err != nil {
		return err
	}

	if navigationDto.Date == nil {
		return nil
	}

	fileName, exist := navigationDto.Date[date]

	if !exist {
		return nil
	}

	err = os.Remove(fileName)

	if err != nil {
		return err
	}

	delete(navigationDto.Date, date)

	content, err = json.Marshal(navigationDto)

	if err != nil {
		return err
	}

	return os.WriteFile(logFileNavigationFile, content, 0644)
}

func (j *JsonStorageProvider) getLogFileByDate(date time.Time) (string, error) {
	content, err := os.ReadFile(logFileNavigationFile)

	if err != nil {
		return "", err
	}

	var navigationDto *models.LogsNavigationDto
	err = json.Unmarshal(content, &navigationDto)

	if err != nil {
		return "", err
	}

	if navigationDto.Date == nil {
		navigationDto.Date = map[string]string{}
	}

	fileName, exist := navigationDto.Date[utils.GetOnlyDate(date)]

	if !exist {
		fileName = fmt.Sprintf(logFilePatternFile, utils.GetOnlyDate(date))

		navigationDto.Date[utils.GetOnlyDate(date)] = fileName
		content, err = json.Marshal(navigationDto)

		if err != nil {
			return "", err
		}

		err = os.WriteFile(logFileNavigationFile, content, 0644)

		if err != nil {
			return "", err
		}
	}

	return func() (string, error) {
		if exist {
			return fileName, nil
		}
		file, err := os.Create(fileName)
		err = file.Close()

		if err != nil {
			return "", err
		}

		if err != nil {
			return "", err
		}

		emptyArray, err := json.Marshal([]models.LogsInfoDto{})

		if err != nil {
			return "", err
		}
		err = os.WriteFile(fileName, emptyArray, 0644)

		if err != nil {
			return "", err
		}

		return fileName, nil
	}()
}
