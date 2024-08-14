package models

type CreateWebhookHistoryRequest struct {
	ChannelID   string `json:"channel_id" validate:"required"`
	WebhookSlug string `json:"webhook_slug"`
	ActionType  string `json:"action_type" validate:"required"`
	StatusCode  string `json:"status_code"`
	EventName   string `json:"event_name"`
	UserName    string `json:"username" validate:"required"`
	Retries     int64  `json:"user_id"`
}

type FeedWebHookRequest struct {
	ChannelID  string `json:"channel_id"`
	EventName  string `json:"event_name"`
	UserName   string `json:"username"`
	ActionType string `json:"action_type"`
	CreatedAt  string `json:"created_at"`
}
