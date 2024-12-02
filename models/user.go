package models

import (
	"logs-aggregator-bot/constants"
	"time"
)

type UserSettingsDto struct {
	UserId        int64
	WorkStarted   time.Time
	CurrentState  constants.UserState
	NeedWorkLogTo time.Time
}
