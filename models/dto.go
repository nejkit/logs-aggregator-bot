package models

type SendNotificationRequest struct {
	ChatId        int64
	Body          string
	Markup        []MarkupData
	IsMultiSelect bool
}

type MarkupData struct {
	Key   string
	Value string
}
