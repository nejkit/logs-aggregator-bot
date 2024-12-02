package models

import "time"

type LogsInfoDto struct {
	Id            string    `json:"id"`
	StartWorkTime time.Time `json:"startWorkTime"`
	EndWorkTime   time.Time `json:"endWorkTime"`
	Message       string    `json:"message"`
}

type LogsNavigationDto struct {
	Date map[string]string `json:"date"`
}
