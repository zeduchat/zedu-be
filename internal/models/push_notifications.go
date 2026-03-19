package models

type PushRequest struct {
	ChannelId   string `json:"channel_id"`
	OrgId       string `json:"org_id"`
	UserId      string `json:"user_id"`
	Message     string `json:"message"`
	ChannelName string `json:"channel_name"`
	TimeStamp   string `json:"time_stamp"`
	AvatarUrl   string `json:"avatar_url"`
	Username    string `json:"username"`
	UserIds     []string
	Payload     any
	Title       string `json:"title"`
}
type OneSignalPushRequest struct {
	Title     string `json:"title" validate:"required"`
	Message   string `json:"message" validate:"required"`
	AvatarUrl string `json:"avatar_url"`
}
