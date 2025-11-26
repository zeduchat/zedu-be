package models

type AgoraTokenRequest struct {
	ChannelName   string `json:"channel_name" validate:"required"`
	Role          string `json:"role" validate:"required,oneof=publisher subscriber"`
	TokenType     string `json:"token_type" validate:"omitempty,oneof=uid userAccount"`
	UID           string `json:"uid,omitempty"`
	ExpirySeconds int    `json:"expiry_seconds" validate:"omitempty,min=60,max=86400"`
}

type AgoraTokenResponse struct {
	RTCToken string `json:"rtc_token,omitempty"`
	RTMToken string `json:"rtm_token,omitempty"`
}
