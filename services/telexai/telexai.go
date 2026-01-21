package telexai

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/openrouter"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func RespondToChat(w http.ResponseWriter, db *storage.Database, logger *utility.Logger, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest, ids models.IDS) (map[string]any, int, error) {
	client := openrouter.GetClient()
	if client == nil {
		logger.Error("OpenRouter client not initialized")
		return map[string]any{}, http.StatusInternalServerError, http.ErrServerClosed
	}

	return client.RespondToChat(w, db, req, extReq, ids)
}

func StreamChatCompletions(w http.ResponseWriter, db *storage.Database, logger *utility.Logger, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest, ids models.IDS) error {
	client := openrouter.GetClient()
	if client == nil {
		logger.Error("OpenRouter client not initialized")
		return http.ErrServerClosed
	}

	return client.StreamChatCompletions(w, db, req, extReq, ids)
}

func ChatCompletions(w http.ResponseWriter, db *storage.Database, logger *utility.Logger, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest, ids models.IDS) (map[string]any, int, error) {
	client := openrouter.GetClient()
	if client == nil {
		logger.Error("OpenRouter client not initialized")
		return map[string]any{}, http.StatusInternalServerError, http.ErrServerClosed
	}

	return client.ChatCompletions(db, req, extReq, ids)
}

func ListModels(logger *utility.Logger, extReq request.ExternalRequest, redisClient *redis.Client, fetchTools bool) (external_models.OpenRouterModelsResponse, error) {
	client := openrouter.GetClient()
	if client == nil {
		logger.Error("OpenRouter client not initialized")
		return external_models.OpenRouterModelsResponse{}, http.ErrServerClosed
	}

	return client.ListModels(extReq, fetchTools)
}

func TranslatorCompletions(logger *utility.Logger, extReq request.ExternalRequest, req models.TelexAIChatCompletionsReq) (map[string]any, int, error) {
	client := openrouter.GetClient()
	if client == nil {
		logger.Error("OpenRouter client not initialized")
		return map[string]any{}, http.StatusInternalServerError, http.ErrServerClosed
	}

	return client.TranslatorCompletions(req, extReq)
}

func ExtractModel(c *gin.Context, logger *utility.Logger, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest, redis *redis.Client) (string, error) {
	client := openrouter.GetClient()
	if client == nil {
		logger.Error("OpenRouter client not initialized")
		return "", http.ErrServerClosed
	}

	return client.ExtractModel(c, req, extReq)
}
