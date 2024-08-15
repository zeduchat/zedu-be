package models

type SlackTelex struct {
	ID          string `gorm:"type:uuid;primary_key" json:"id"`
	UserID      string `json:"user_id" gorm:"type:uuid;not null"`
	OauthCode   string `json:"oauth_code,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	Message     string `json:"Message,omitempty"`
	Channel     string `json:"channel,omitempty"`
}
