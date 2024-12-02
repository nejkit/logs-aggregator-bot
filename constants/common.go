package constants

const (
	CallbackParamContinueOldLog = "continue_old_log"
	CallbackParamCreateNewLog   = "create_new_log"
)

type UserState int

const (
	UserStateNone UserState = iota
	UserStateSelectLogType
	UserStateSelectOldLogDate
	UserStateSelectNewLogDate
	UserStateSelectNewLogMessage
)

type Commands string

const (
	StartWorkDayCommand Commands = "start_work_day"
	EndWorkDayCommand   Commands = "end_work_day"
	GetLogsCommand      Commands = "get_today_logs"
)
