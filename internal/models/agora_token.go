package models

// HuddleAgoraTokenRequest is the request to get an Agora RTC token for a huddle
type HuddleAgoraTokenRequest struct {
	HuddleID string `json:"huddle_id" validate:"required,uuid"`
}

// HuddleAgoraTokenResponse contains the Agora RTC token and connection details for a huddle
type HuddleAgoraTokenResponse struct {
	Token       string `json:"token"`
	AppId       string `json:"app_id"`
	ChannelName string `json:"channel_name"`
	UID         uint32 `json:"uid"`
}
