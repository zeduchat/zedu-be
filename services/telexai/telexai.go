package telexai

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	rd "github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/utility"
)

func ChatCompletions(db *storage.Database, logger *utility.Logger, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest, ids models.IDS) (map[string]any, int, error) {
	var (
		telexlogs models.TelexAIUsageLog
	)

	openRouterPayload := external_models.OpenRouterReq{
		Model:    req.GetModel(),
		Messages: req.Messages,
		ExtraBody: external_models.OpenRouterExtraBody{
			Usage: external_models.OpenRouterUsageToggle{
				Include: true,
			},
		},
		Tools: ConvertTools(req.Tools),
	}

	logger.Info(fmt.Sprintf("Making request to model: %s for org: %s", req.GetModel(), ids.OrganisationID))
	res, err := extReq.SendExternalRequest(request.GetChatCompletions, openRouterPayload)
	if err != nil {
		if strings.Contains(err.Error(), "429") {
			logger.Error("OpenRouter API call failed with 429: ", err)
			return map[string]any{}, http.StatusTooManyRequests, fmt.Errorf("rate limit exceeded: %w", err)
		}
		logger.Error("OpenRouter API call failed: ", err)
		return map[string]any{}, http.StatusBadRequest, err
	}

	result, ok := res.(map[string]any)
	if !ok {
		logger.Error("failed to get chat completions: ", res)
		return map[string]any{}, http.StatusBadRequest, fmt.Errorf("failed to get chat completions: %v", res)
	}

	if choices, exists := result["choices"].([]any); !exists || len(choices) == 0 {
		return map[string]any{}, http.StatusBadRequest, fmt.Errorf("no choices found in response")
	}

	if usage, exists := result["usage"].(external_models.OpenRouterUsage); exists {
		if err := telexlogs.CreateUsageLog(db.Postgresql, logger, ids, req, usage); err != nil {
			logger.Error("failed to create usage log: ", err)
			return map[string]any{}, http.StatusBadRequest, fmt.Errorf("failed to create usage log: %w", err)
		}
	}

	return result, http.StatusOK, nil
}

func ListModels(logger *utility.Logger, extReq request.ExternalRequest, redisClient *redis.Client, fetchTools bool) (external_models.OpenRouterModelsResponse, error) {
	var redisKey string

	cacheDuration := 12 * time.Hour
	if fetchTools {
		redisKey = "telexai:tools_models"
	}
	redisKey = "telexai:models"

	cachedModels, err := rd.RedisGet(redisClient, redisKey)
	if err == nil && len(cachedModels) > 0 {
		logger.Info("Using cached models from Redis")

		var rawJSON string
		if err := json.Unmarshal([]byte(cachedModels), &rawJSON); err != nil {
			logger.Error("Failed to unmarshal outer JSON: ", err)
			return external_models.OpenRouterModelsResponse{}, fmt.Errorf("failed to unmarshal outer JSON: %w", err)
		}

		var cachedModelsList external_models.OpenRouterModelsResponse
		if err := json.Unmarshal([]byte(rawJSON), &cachedModelsList); err != nil {
			logger.Error("Failed to unmarshal inner JSON: ", err)
			return external_models.OpenRouterModelsResponse{}, fmt.Errorf("failed to unmarshal inner JSON: %w", err)
		}
		return cachedModelsList, nil
	}

	logger.Info("No cached models found in Redis, fetching from OpenRouter")
	res, err := extReq.SendExternalRequest(request.GetAllModels, fetchTools)
	if err != nil {
		logger.Error("Failed to fetch models: ", err)
		return external_models.OpenRouterModelsResponse{}, fmt.Errorf("failed to fetch models: %w", err)
	}
	modelsList, ok := res.(external_models.OpenRouterModelsResponse)
	if !ok {
		logger.Error("Invalid response format for models")
		return external_models.OpenRouterModelsResponse{}, fmt.Errorf("invalid response format for models")
	}

	modelsJSON, err := json.Marshal(modelsList)
	if err != nil {
		logger.Error("Failed to marshal models for caching: ", err)
	} else {
		if err := rd.RedisSet(redisClient, redisKey, string(modelsJSON), cacheDuration); err != nil {
			logger.Error("Failed to cache models in Redis: ", err)
		} else {
			logger.Info("Successfully cached models in Redis")
		}
	}

	return modelsList, nil
}

func ExtractModel(c *gin.Context, logger *utility.Logger, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest, redis *redis.Client) (string, error) {
	var availableModels external_models.OpenRouterModelsResponse

	if req.Tools != nil {
		availableModels, _ = ListModels(logger, extReq, redis, true)
	}
	availableModels, _ = ListModels(logger, extReq, redis, false)

	models := availableModels.Data
	modelMap := make(map[string]bool)
	for _, model := range models {
		modelMap[model.ID] = true
	}

	var selectedModel string
	if headerModel := c.GetHeader("X-Model"); headerModel != "" {
		logger.Info("Model selected via header: ", headerModel)
		selectedModel = strings.TrimSpace(headerModel)
	} else if queryModel := c.Query("model"); queryModel != "" {
		logger.Info("Model selected via query: ", queryModel)
		selectedModel = strings.TrimSpace(queryModel)
	} else if req.Model != "" {
		logger.Info("Model selected via body: ", req.Model)
		selectedModel = strings.TrimSpace(req.Model)
	} else {
		logger.Info("Using default model")
		selectedModel = "deepseek/deepseek-r1-0528-qwen3-8b:free"
	}

	if _, exists := modelMap[selectedModel]; !exists {
		logger.Error("Invalid model selected: ", selectedModel)
		return "deepseek/deepseek-r1-0528-qwen3-8b:free", fmt.Errorf("invalid model selected: %s", selectedModel)
	}

	return selectedModel, nil
	// return "deepseek/deepseek-r1-0528-qwen3-8b:free", nil
}

func ChargeAICreditUsage(db *storage.Database, ids models.IDS, inputputLength int, logger *utility.Logger) error {
	var agentPrice float64 = 0.0 // temp value

	creditUsed := models.CalculateCreditCost(inputputLength, agentPrice)

	credit_usage := models.CreditUsage{
		ID:             utility.GenerateUUID(),
		OrganisationID: ids.OrganisationID,
		Amount:         creditUsed,
		AgentID:        ids.AgentID,
		UserID:         nil,
	}

	err := credit_usage.UpdateOrCreateDailyCredit(db.Postgresql, creditUsed)
	if err != nil {
		logger.Error("failed to create credit usage!!")
	}

	if err = models.UpdateOrgCreditBalance(db.Postgresql, ids.OrganisationID); err != nil {
		logger.Error("Organisation credit Recalculation failed")
	}

	return nil
}

func ExtractChatContent(response map[string]any) (string, error) {
	choices, ok := response["choices"].([]any)
	if !ok {
		return "", errors.New("invalid or empty choices")
	}

	firstChoice, ok := choices[0].(map[string]any)
	if !ok {
		return "", errors.New("invalid choice format")
	}

	message, ok := firstChoice["message"].(map[string]any)
	if !ok {
		return "", errors.New("invalid message format")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", errors.New("content not found or not a string")
	}

	return content, nil
}
