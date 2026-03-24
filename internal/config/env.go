package config

import (
	"encoding/json"
	"os"
)

type Configuration struct {
	Server       ServerConfiguration
	Database     Database
	TestDatabase Database
	App          App
	IPStack      IPStack
	Centrifuge   Centrifuge
	Redis        Redis
	Mail         MAIL
	Minio        Minio
	Slack        Slack
	Stripe       Stripe
	TypeSense    TypeSense
	Channels     Channels
	RabbitMQ     RabbitMQ
	Elastic      ElasticDb
	Firebase     Firebase
	OneSignal    OneSignal
	MongoDB      MongoDB
	OpenRouter   OpenRouter
	Admin        Admin
	Google       Google
	Apple        Apple
	WebPush      WebPush
	Agora        Agora
}

type BaseConfig struct {
	SERVER_PORT                      string  `mapstructure:"SERVER_PORT"`
	SERVER_SECRET                    string  `mapstructure:"SERVER_SECRET"`
	SERVER_ENCKEY                    string  `mapstructure:"SERVER_ENCKEY"`
	SERVER_ACCESSTOKENEXPIREDURATION int     `mapstructure:"SERVER_ACCESSTOKENEXPIREDURATION"`
	REQUEST_PER_SECOND               float64 `mapstructure:"REQUEST_PER_SECOND"`
	TRUSTED_PROXIES                  string  `mapstructure:"TRUSTED_PROXIES"`
	EXEMPT_FROM_THROTTLE             string  `mapstructure:"EXEMPT_FROM_THROTTLE"`
	ENCRYPTION_KEY                   string  `mapstructure:"ENCRYPTION_KEY"`

	APP_NAME                string `mapstructure:"APP_NAME"`
	APP_MODE                string `mapstructure:"APP_MODE"`
	APP_URL                 string `mapstructure:"APP_URL"`
	MAGIC_LINK_DURATION     int    `mapstructure:"MAGIC_LINK_DURATION"`
	RESET_PASSWORD_DURATION int    `mapstructure:"RESET_PASSWORD_DURATION"`
	WEBHOOK_API_URL         string `mapstructure:"WEBHOOK_API_URL"`
	FRONTEND_URL            string `mapstructure:"FRONTEND_URL"`

	DB_HOST       string `mapstructure:"DB_HOST"`
	DB_PORT       string `mapstructure:"DB_PORT"`
	DB_CONNECTION string `mapstructure:"DB_CONNECTION"`
	TIMEZONE      string `mapstructure:"TIMEZONE"`
	SSLMODE       string `mapstructure:"SSLMODE"`
	USERNAME      string `mapstructure:"USERNAME"`
	PASSWORD      string `mapstructure:"PASSWORD"`
	DB_NAME       string `mapstructure:"DB_NAME"`
	MIGRATE       bool   `mapstructure:"MIGRATE"`

	TEST_DB_HOST       string `mapstructure:"TEST_DB_HOST"`
	TEST_DB_PORT       string `mapstructure:"TEST_DB_PORT"`
	TEST_DB_CONNECTION string `mapstructure:"TEST_DB_CONNECTION"`
	TEST_TIMEZONE      string `mapstructure:"TEST_TIMEZONE"`
	TEST_SSLMODE       string `mapstructure:"TEST_SSLMODE"`
	TEST_USERNAME      string `mapstructure:"TEST_USERNAME"`
	TEST_PASSWORD      string `mapstructure:"TEST_PASSWORD"`
	TEST_DB_NAME       string `mapstructure:"TEST_DB_NAME"`
	TEST_MIGRATE       bool   `mapstructure:"TEST_MIGRATE"`

	IPSTACK_KEY      string `mapstructure:"IPSTACK_KEY"`
	IPSTACK_BASE_URL string `mapstructure:"IPSTACK_BASE_URL"`

	HMAC_SECRET        string `mapstructure:"HMAC_SECRET"`
	CENTRIFUGE_URL     string `mapstructure:"CENTRIFUGE_URL"`
	CENTRIFUGE_API_KEY string `mapstructure:"CENTRIFUGE_API_KEY"`

	MAIL_SERVER   string `mapstructure:"MAIL_SERVER"`
	MAIL_PASSWORD string `mapstructure:"MAIL_PASSWORD"`
	MAIL_USERNAME string `mapstructure:"MAIL_USERNAME"`
	MAIL_PORT     string `mapstructure:"MAIL_PORT"`

	REDIS_PORT string `mapstructure:"REDIS_PORT"`
	REDIS_HOST string `mapstructure:"REDIS_HOST"`
	REDIS_DB   string `mapstructure:"REDIS_DB"`

	MINIO_ENDPOINT    string `mapstructure:"MINIO_ENDPOINT"`
	BUCKET_NAME       string `mapstructure:"BUCKET_NAME"`
	BUCKET_ACCESS_KEY string `mapstructure:"BUCKET_ACCESS_KEY"`
	BUKCET_SECRET_KEY string `mapstructure:"BUKCET_SECRET_KEY"`
	MINIO_USE_SSL     bool   `mapstructure:"MINIO_USE_SSL"`

	SLACK_CLIENT_ID     string `mapstructure:"SLACK_CLIENT_ID"`
	SLACK_CLIENT_SECRET string `mapstructure:"SLACK_CLIENT_SECRET"`
	SLACK_REDIRECT_URI  string `mapstructure:"SLACK_REDIRECT_URI"`
	SLACK_BASE_URL      string `mapstructure:"SLACK_BASE_URL"`
	SLACK_MANIFEST_URL  string `mapstructure:"SLACK_MANIFEST_URL"`
	SLACK_APP_ID        string `mapstructure:"SLACK_APP_ID"`
	SLACK_REFRESH_TOKEN string `mapstructure:"SLACK_REFRESH_TOKEN"`

	TYPESENSE_API_URL string `mapstructure:"TYPESENSE_API_URL"`
	TYPESENSE_API_KEY string `mapstructure:"TYPESENSE_API_KEY"`

	STRIPE_KEY            string `mapstructure:"STRIPE_KEY"`
	STRIPE_WEBHOOK_SECRET string `mapstructure:"STRIPE_WEBHOOK_SECRET"`
	STRIPE_BASIC_ID       string `mapstructure:"STRIPE_BASIC_ID"`
	STRIPE_PREMIUM_ID     string `mapstructure:"STRIPE_PREMIUM_ID"`
	STRIPE_ADVANCED_ID    string `mapstructure:"STRIPE_ADVANCED_ID"`

	STRIPE_BASIC_CREDIT_ID    string `mapstructure:"STRIPE_BASIC_CREDIT_ID"`
	STRIPE_PREMIUM_CREDIT_ID  string `mapstructure:"STRIPE_PREMIUM_CREDIT_ID"`
	STRIPE_ADVANCED_CREDIT_ID string `mapstructure:"STRIPE_ADVANCED_CREDIT_ID"`

	TELEX_LOGIN_CHANNEL  string `mapstructure:"TELEX_LOGIN_CHANNEL"`
	TELEX_SIGNUP_CHANNEL string `mapstructure:"TELEX_SIGNUP_CHANNEL"`

	RABBITMQ_CONNECTION string `mapstructure:"RABBITMQ_CONNECTION"`
	RABBITMQ_EXCHANGE   string `mapstructure:"RABBITMQ_EXCHANGE"`

	ELASTIC_URL                string `mapstructure:"ELASTIC_URL"`
	ELASTIC_API_KEY            string `mapstructure:"ELASTIC_API_KEY"`
	FIREBASE_SERVICE_FILE_PATH string `mapstructure:"FIREBASE_SERVICE_FILE_PATH"`

	OPENROUTER_API_KEY  string `mapstructure:"OPENROUTER_API_KEY"`
	OPENROUTER_BASE_URL string `mapstructure:"OPENROUTER_BASE_URL"`

	MONGO_URI      string `mapstructure:"MONGO_URI"`
	MONGO_DB_NAME  string `mapstructure:"MONGO_DB_NAME"`
	MONGO_DISABLED bool   `mapstructure:"MONGO_DISABLED"`

	SUPER_ADMIN_EMAIL    string `mapstructure:"SUPER_ADMIN_EMAIL"`
	SUPER_ADMIN_NAME     string `mapstructure:"SUPER_ADMIN_NAME"`
	SUPER_ADMIN_PASSWORD string `mapstructure:"SUPER_ADMIN_PASSWORD"`
	SUPER_ADMIN_ROLE     string `mapstructure:"SUPER_ADMIN_ROLE"`

	GOOGLE_CLIENT_ID     string `mapstructure:"GOOGLE_CLIENT_ID"`
	GOOGLE_CLIENT_SECRET string `mapstructure:"GOOGLE_CLIENT_SECRET"`
	GOOGLE_REDIRECT_URI  string `mapstructure:"GOOGLE_REDIRECT_URI"`
	APPLE_CLIENT_ID      string `mapstructure:"APPLE_CLIENT_ID"`
	APPLE_TEAM_ID        string `mapstructure:"APPLE_TEAM_ID"`
	APPLE_KEY_ID         string `mapstructure:"APPLE_KEY_ID"`
	APPLE_PRIVATE_KEY    string `mapstructure:"APPLE_PRIVATE_KEY"`
	PubKey               string `mapstructure:"NEXT_PUBLIC_VAPID_PUBLIC_KEY"`
	PrivKey              string `mapstructure:"NEXT_PUBLIC_VAPID_PRIVATE_KEY"`

	AGORA_APP_ID          string `mapstructure:"AGORA_APP_ID"`
	AGORA_APP_CERTIFICATE string `mapstructure:"AGORA_APP_CERTIFICATE"`
	AGORA_CUSTOMER_ID     string `mapstructure:"AGORA_CUSTOMER_ID"`
	AGORA_CUSTOMER_SECRET string `mapstructure:"AGORA_CUSTOMER_SECRET"`

	ONESIGNAL_APP_ID       string `mapstructure:"ONESIGNAL_APP_ID"`
	ONESIGNAL_REST_API_KEY string `mapstructure:"ONESIGNAL_REST_API_KEY"`
	ONESIGNAL_ENABLED      bool   `mapstructure:"ONESIGNAL_ENABLED"`
}

func (config *BaseConfig) SetupConfigurationn() *Configuration {
	trustedProxies := []string{}
	exemptFromThrottle := []string{}
	json.Unmarshal([]byte(config.TRUSTED_PROXIES), &trustedProxies)
	json.Unmarshal([]byte(config.EXEMPT_FROM_THROTTLE), &exemptFromThrottle)
	if config.SERVER_PORT == "" {
		config.SERVER_PORT = os.Getenv("PORT")
	}
	return &Configuration{
		Server: ServerConfiguration{
			Port:                      config.SERVER_PORT,
			Secret:                    config.SERVER_SECRET,
			EncKey:                    config.SERVER_ENCKEY,
			AccessTokenExpireDuration: config.SERVER_ACCESSTOKENEXPIREDURATION,
			RequestPerSecond:          config.REQUEST_PER_SECOND,
			TrustedProxies:            trustedProxies,
			ExemptFromThrottle:        exemptFromThrottle,
			EncryptionKey:             config.ENCRYPTION_KEY,
		},
		App: App{
			Name:                  config.APP_NAME,
			Mode:                  config.APP_MODE,
			Url:                   config.APP_URL,
			MagicLinkDuration:     config.MAGIC_LINK_DURATION,
			ResetPasswordDuration: config.RESET_PASSWORD_DURATION,
			WebhookApiUrl:         config.WEBHOOK_API_URL,
			FRONTEND_URL:          config.FRONTEND_URL,
			OpenRouterApiKey:      config.OPENROUTER_API_KEY,
		},
		Database: Database{
			DB_HOST:       config.DB_HOST,
			DB_PORT:       config.DB_PORT,
			DB_CONNECTION: config.DB_CONNECTION,
			USERNAME:      config.USERNAME,
			PASSWORD:      config.PASSWORD,
			TIMEZONE:      config.TIMEZONE,
			SSLMODE:       config.SSLMODE,
			DB_NAME:       config.DB_NAME,
			Migrate:       config.MIGRATE,
		},
		TestDatabase: Database{
			DB_HOST:       config.TEST_DB_HOST,
			DB_PORT:       config.TEST_DB_PORT,
			DB_CONNECTION: config.TEST_DB_CONNECTION,
			USERNAME:      config.TEST_USERNAME,
			PASSWORD:      config.TEST_PASSWORD,
			TIMEZONE:      config.TEST_TIMEZONE,
			SSLMODE:       config.TEST_SSLMODE,
			DB_NAME:       config.TEST_DB_NAME,
			Migrate:       config.TEST_MIGRATE,
		},

		IPStack: IPStack{
			Key:     config.IPSTACK_KEY,
			BaseUrl: config.IPSTACK_BASE_URL,
		},

		Centrifuge: Centrifuge{
			Secret: config.HMAC_SECRET,
			Url:    config.CENTRIFUGE_URL,
			ApiKey: config.CENTRIFUGE_API_KEY,
		},

		Mail: MAIL{
			Server:   config.MAIL_SERVER,
			Password: config.MAIL_PASSWORD,
			Port:     config.MAIL_PORT,
			Username: config.MAIL_USERNAME,
		},

		Redis: Redis{
			REDIS_PORT: config.REDIS_PORT,
			REDIS_HOST: config.REDIS_HOST,
			REDIS_DB:   config.REDIS_DB,
		},

		Minio: Minio{
			MinioEndpoint: config.MINIO_ENDPOINT,
			BucketName:    config.BUCKET_NAME,
			AccessKey:     config.BUCKET_ACCESS_KEY,
			Secret:        config.BUKCET_SECRET_KEY,
			UseSSL:        config.MINIO_USE_SSL,
		},

		Slack: Slack{
			ClientId:     config.SLACK_CLIENT_ID,
			ClientSecret: config.SLACK_CLIENT_SECRET,
			RedirectURI:  config.SLACK_REDIRECT_URI,
			BaseUrl:      config.SLACK_BASE_URL,
			ManifestUrl:  config.SLACK_MANIFEST_URL,
			AppId:        config.SLACK_APP_ID,
			RefreshToken: config.SLACK_REFRESH_TOKEN,
		},

		Stripe: Stripe{
			STRIPE_KEY:                config.STRIPE_KEY,
			STRIPE_WEBHOOK_SECRET:     config.STRIPE_WEBHOOK_SECRET,
			STRIPE_BASIC_ID:           config.STRIPE_BASIC_ID,
			STRIPE_PREMIUM_ID:         config.STRIPE_PREMIUM_ID,
			STRIPE_ADVANCED_ID:        config.STRIPE_ADVANCED_ID,
			STRIPE_BASIC_CREDIT_ID:    config.STRIPE_BASIC_CREDIT_ID,
			STRIPE_PREMIUM_CREDIT_ID:  config.STRIPE_PREMIUM_CREDIT_ID,
			STRIPE_ADVANCED_CREDIT_ID: config.STRIPE_ADVANCED_CREDIT_ID,
		},

		TypeSense: TypeSense{
			TypeSense_API_URL: config.TYPESENSE_API_URL,
			TypeSense_API_KEY: config.TYPESENSE_API_KEY,
		},

		Channels: Channels{
			Login:  config.TELEX_LOGIN_CHANNEL,
			Signup: config.TELEX_SIGNUP_CHANNEL,
		},
		RabbitMQ: RabbitMQ{
			Connection: config.RABBITMQ_CONNECTION,
			Exchange:   config.RABBITMQ_EXCHANGE,
		},
		Elastic: ElasticDb{
			ElasticEndpoint: config.ELASTIC_URL,
			ElasticApiKey:   config.ELASTIC_API_KEY,
		},
		Firebase: Firebase{
			ServiceFilePath: config.FIREBASE_SERVICE_FILE_PATH,
		},
		MongoDB: MongoDB{
			Mongo_URI:      config.MONGO_URI,
			DB_Name:        config.MONGO_DB_NAME,
			Mongo_Disabled: config.MONGO_DISABLED,
		},
		OpenRouter: OpenRouter{
			ApiKey:  config.OPENROUTER_API_KEY,
			BaseUrl: config.OPENROUTER_BASE_URL,
		},
		Admin: Admin{
			SUPER_ADMIN_EMAIL:    config.SUPER_ADMIN_EMAIL,
			SUPER_ADMIN_NAME:     config.SUPER_ADMIN_NAME,
			SUPER_ADMIN_PASSWORD: config.SUPER_ADMIN_PASSWORD,
			SUPER_ADMIN_ROLE:     config.SUPER_ADMIN_ROLE,
		},
		Google: Google{
			CLIENT_ID:     config.GOOGLE_CLIENT_ID,
			CLIENT_SECRET: config.GOOGLE_CLIENT_SECRET,
			REDIRECT_URI:  config.GOOGLE_REDIRECT_URI,
		},
		Apple: Apple{
			CLIENT_ID:   config.APPLE_CLIENT_ID,
			TEAM_ID:     config.APPLE_TEAM_ID,
			KEY_ID:      config.APPLE_KEY_ID,
			PRIVATE_KEY: config.APPLE_PRIVATE_KEY,
		},
		WebPush: WebPush{
			PubKey:  config.PubKey,
			PrivKey: config.PrivKey,
		},
		Agora: Agora{
			AppId:          config.AGORA_APP_ID,
			AppCertificate: config.AGORA_APP_CERTIFICATE,
			CustomerID:     config.AGORA_CUSTOMER_ID,
			CustomerSecret: config.AGORA_CUSTOMER_SECRET,
		},
		OneSignal: OneSignal{
			AppID:      config.ONESIGNAL_APP_ID,
			RestAPIKey: config.ONESIGNAL_REST_API_KEY,
			Enabled:    config.ONESIGNAL_ENABLED,
		},
	}
}
