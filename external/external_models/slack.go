package external_models

type SlackOAuthResponse struct {
	AccessToken string `json:"access_token"`
	Team        struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
	IncomingWebHook struct {
		Channel          string `json:"channel"`
		ChannelId        string `json:"channel_id"`
		ConfigurationUrl string `json:"configuration_url"`
		Url              string `json:"url"`
	} `json:"incoming_webhook"`
	Error string `json:"error,omitempty"`
}

type SlackChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SlackChannelResponse struct {
	Ok       bool           `json:"ok"`
	Channels []SlackChannel `json:"channels"`
	Error    string         `json:"error,omitempty"`
}

type SlackTokenResponse map[string]any

type SlackTokenOutput struct {
	Ok           bool   `json:"ok"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	TeamID       string `json:"team_id"`
	UserID       string `json:"user_id"`
	IAT          int64  `json:"iat"` // Issued At timestamp
	Exp          int64  `json:"exp"` // Expiration timestamp
}

type SlackManifestResponse map[string]any
