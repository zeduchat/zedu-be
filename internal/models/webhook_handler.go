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
	OrgID       string `json:"org_id"`
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
	Content    string `json:"message"`
}

type FeedQueue struct {
	ChannelsId string                 `json:"channel_id"`
	ThreadId   string                 `json:"thread_id"`
	AgentName  string                 `json:"agent_name"`
	UserId     string                 `json:"user_id"`
	ReturnUrl  string                 `json:"return_url"`
	Content    string                 `json:"message"`
	Type       string                 `json:"type"`
	OrgId      string                 `json:"org_id"`
	Media      []UploadedFileResponse `json:"media"`
	Mentions   []Mention              `json:"mentions"`
}

type QueueFeed struct {
	ChannelsId string             `json:"channel_id"`
	ThreadId   string             `json:"thread_id"`
	UserId     string             `json:"user_id"`
	ReturnUrl  string             `json:"return_url"`
	Content    FeedWebHookRequest `json:"message_content"`
	Type       string             `json:"type"`
	OrgID      string             `json:"org_id"`
}
