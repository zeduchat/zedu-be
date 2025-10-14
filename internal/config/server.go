package config

type ServerConfiguration struct {
	Port                      string
	Secret                    string
	EncKey                    string
	AccessTokenExpireDuration int
	RequestPerSecond          float64
	TrustedProxies            []string
	ExemptFromThrottle        []string
	EncryptionKey             string
}

type App struct {
	Name                  string
	Mode                  string
	Url                   string
	MagicLinkDuration     int
	ResetPasswordDuration int
	WebhookApiUrl         string
	FRONTEND_URL          string
	OpenRouterApiKey      string
}
