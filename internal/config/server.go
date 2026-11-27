package config

import "strings"

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

// FrontendBaseURL returns FRONTEND_URL with exactly one trailing slash.
// Use it for every user-facing link; never use request headers.
func FrontendBaseURL() string {
	cfg := GetConfig()
	if cfg == nil {
		return ""
	}
	return strings.TrimRight(cfg.App.FRONTEND_URL, "/") + "/"
}
