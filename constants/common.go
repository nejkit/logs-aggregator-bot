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
