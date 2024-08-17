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
