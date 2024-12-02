package constants

const (
	CallbackParamContinueOldLog = "continue_old_log"
	CallbackParamCreateNewLog   = "create_new_log"
	CallbackStopDeleteLogs      = "stop_delete_log"
)

type UserState int

const (
	UserStateNone UserState = iota
	UserStateSelectLogType
	UserStateSelectOldLogDate
	UserStateSelectNewLogDate
	UserStateSelectNewLogMessage
	UserStateSelectLogDate
	UserStateSelectLogsToDelete
)

type Commands string

const (
	StartWorkDayCommand Commands = "start_work_day"
	EndWorkDayCommand   Commands = "end_work_day"
	GetLogsCommand      Commands = "get_today_logs"
	GetAllLogsCommand   Commands = "get_all_logs"
	DeleteLogsCommand   Commands = "delete_logs_command"
)
