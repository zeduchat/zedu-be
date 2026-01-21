package openrouter

import (
	"github.com/go-redis/redis/v8"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

type Client struct {
	ApiKey      string
	BaseUrl     string
	Logger      *utility.Logger
	RedisClient *redis.Client
}

var OpenRouterClient *Client

func NewOpenRouterClient(logger *utility.Logger, cfg config.OpenRouter, redisClient *redis.Client) *Client {
	client := &Client{
		ApiKey:      cfg.ApiKey,
		BaseUrl:     cfg.BaseUrl,
		Logger:      logger,
		RedisClient: redisClient,
	}

	OpenRouterClient = client
	utility.LogAndPrint(logger, "OpenRouter client initialized")
	return client
}

func GetClient() *Client {
	return OpenRouterClient
}
