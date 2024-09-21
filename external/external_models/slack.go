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


type SlackManifestResponse struct {
	Metadata struct {
		MajorVersion int `json:"major_version"`
		MinorVersion int `json:"minor_version"`
	} `json:"_metadata"`
	DisplayInformation struct {
		Name            string `json:"name"`
		LongDescription string `json:"long_description"`
		Description     string `json:"description"`
		BackgroundColor string `json:"background_color"`
	} `json:"display_information"`
	Settings struct {
		SocketModeEnabled bool `json:"socket_mode_enabled"`
		Interactivity     struct {
			IsEnabled   bool   `json:"is_enabled"`
			RequestURL  string `json:"request_url"`
		} `json:"interactivity"`
		EventSubscriptions struct {
			BotEvents []string `json:"bot_events"`
		} `json:"event_subscriptions"`
	} `json:"settings"`
	Features struct {
		AppHome struct {
			HomeTabEnabled           bool `json:"home_tab_enabled"`
			MessagesTabEnabled        bool `json:"messages_tab_enabled"`
			MessagesTabReadOnlyEnabled bool `json:"messages_tab_read_only_enabled"`
		} `json:"app_home"`
		BotUser struct {
			DisplayName string `json:"display_name"`
		} `json:"bot_user"`
		SlashCommands []struct {
			Command     string `json:"command"`
			Description string `json:"description"`
			UsageHint   string `json:"usage_hint"`
			URL         string `json:"url"`
		} `json:"slash_commands"`
	} `json:"features"`
	OAuthConfig struct {
		Scopes struct {
			Bot []string `json:"bot"`
		} `json:"scopes"`
		RedirectURLs []string `json:"redirect_urls"`
	} `json:"oauth_config"`
}