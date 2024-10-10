package models

type CreateWebhookHistoryRequest struct {
	ChannelID   string `json:"channel_id"`
	WebhookSlug string `json:"webhook_slug"`
	ActionType  string `json:"action_type"`
	StatusCode  string `json:"status_code"`
	EventName   string `json:"event_name" validate:"required"`
	UserName    string `json:"username"   validate:"required"`
	Retries     int64  `json:"retries"`
	Status      string `json:"status" validate:"required,oneof=success error"`
	AvatarURL   string `json:"avatar_url"`
	Message     string `json:"message" validate:"required"`
	UserID      string `json:"user_id"`
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

type FeedQueue struct {
	ChannelsId string `json:"channel_id"`
	ThreadId   string `json:"thread_id"`
	UserId     string `json:"user_id"`
	ReturnUrl  string `json:"return_url"`
	Content    string `json:"message"`
	Type       string `json:"type"`
}
