package external_models

type SlackOAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	BotUserID   string `json:"bot_user_id"`
	AppID       string `json:"app_id"`
	Team        struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
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
