package config

type Slack struct {
	ClientId     string
	ClientSecret string
	RedirectURI  string
	BaseUrl      string
	ManifestUrl  string
	AppId        string
	RefreshToken string
}
