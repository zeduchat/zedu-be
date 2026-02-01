package activity

import "time"

type MentionActivityItem struct {
	ThreadID       string    `json:"thread_id"`
	ChannelID      string    `json:"channels_id"`
	ChannelName    string    `json:"channel_name"`
	ChannelType    string    `json:"channel_type"`
	Content        string    `json:"message"`
	SenderID       string    `json:"user_id"`
	SenderUsername string    `json:"username"`
	SenderAvatar   string    `json:"avatar_url"`
	OrganisationID string    `json:"org_id"`
	CreatedAt      time.Time `json:"created_at"`
}
