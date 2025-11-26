package models

// AgoraTokenRequest is the request to get an Agora RTC token
type AgoraTokenRequest struct {
	BuzzID string `json:"buzz_id" validate:"required,uuid"`
}

// AgoraTokenResponse contains the Agora RTC token and connection details
type AgoraTokenResponse struct {
	Token       string `json:"token"`
	AppId       string `json:"app_id"`
	ChannelName string `json:"channel_name"`
	UID         uint32 `json:"uid"`
}
