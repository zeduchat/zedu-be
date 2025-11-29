package models

// BuzzAgoraTokenRequest is the request to get an Agora RTC token for a buzz
type BuzzAgoraTokenRequest struct {
	BuzzID string `json:"buzz_id" validate:"required,uuid"`
	UID    string `json:"uid" validate:"required,uuid"`
}

// BuzzAgoraTokenResponse contains the Agora RTC token and connection details for a buzz
type BuzzAgoraTokenResponse struct {
	Token       string `json:"token"`
	AppId       string `json:"app_id"`
	ChannelName string `json:"channel_name"`
	UID         string `json:"uid"`
}
