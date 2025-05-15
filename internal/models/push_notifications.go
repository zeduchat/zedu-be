package models

type PushFCMRequest struct {
	ChannelId   string `json:"channel_id"`
	UserId      string `json:"user_id"`
	Message     string `json:"message"`
	ChannelName string `json:"channel_name"`
	TimeStamp   string `json:"time_stamp"`
	AvatarUrl   string `json:"avatar_url"`
	Username    string `json:"username"`
	UserIds     []string
}
