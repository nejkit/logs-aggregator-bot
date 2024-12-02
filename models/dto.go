package models

type SendNotificationRequest struct {
	ChatId int64
	Body   string
	Markup []MarkupData
}

type MarkupData struct {
	Key   string
	Value string
}
