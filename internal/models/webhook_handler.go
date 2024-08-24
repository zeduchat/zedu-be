package models

type CreateWebhookHistoryRequest struct {
	ChannelID   string `json:"channel_id"`
	WebhookSlug string `json:"webhook_slug"`
	ActionType  string `json:"action_type" validate:"required"`
	StatusCode  string `json:"status_code"`
	EventName   string `json:"event_name" validate:"required"`
	UserName    string `json:"username"`
	Retries     int64  `json:"retries"`
	Status      string `json:"status" validate:"required,oneof=success error"`
	AvatarURL   string `json:"avatar_url"`
	Content     string `json:"content"`
}

type FeedWebHookRequest struct {
	ChannelID  string `json:"channel_id"`
	EventName  string `json:"event_name"`
	UserName   string `json:"username"`
	ActionType string `json:"action_type"`
	CreatedAt  string `json:"created_at"`
	Status     string `json:"status"`
	AvatarURL  string `json:"avatar_url,omitempty"`
	Type       string `json:"type"`
	Content    string `json:"content"`
}
