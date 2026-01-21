package openrouter

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	rd "github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/utility"
)

func (c *Client) ListModels(extReq request.ExternalRequest, fetchTools bool) (external_models.OpenRouterModelsResponse, error) {
	var redisKey string

	cacheDuration := 12 * time.Hour
	if fetchTools {
		redisKey = "telexai:tools_models"
	} else {
		redisKey = "telexai:models"
	}

	cachedModels, err := rd.RedisGet(c.RedisClient, redisKey)
	if err == nil && len(cachedModels) > 0 {
		c.Logger.Info("Using cached models from Redis")

		var rawJSON string
		if err := json.Unmarshal([]byte(cachedModels), &rawJSON); err != nil {
			c.Logger.Error("Failed to unmarshal outer JSON: ", err)
			return external_models.OpenRouterModelsResponse{}, fmt.Errorf("failed to unmarshal outer JSON: %w", err)
		}

		var cachedModelsList external_models.OpenRouterModelsResponse
		if err := json.Unmarshal([]byte(rawJSON), &cachedModelsList); err != nil {
			c.Logger.Error("Failed to unmarshal inner JSON: ", err)
			return external_models.OpenRouterModelsResponse{}, fmt.Errorf("failed to unmarshal inner JSON: %w", err)
		}
		return cachedModelsList, nil
	}

	c.Logger.Info("No cached models found in Redis, fetching from OpenRouter")
	res, err := extReq.SendExternalRequest(request.GetAllModels, fetchTools)
	if err != nil {
		c.Logger.Error("Failed to fetch models: ", err)
		return external_models.OpenRouterModelsResponse{}, fmt.Errorf("failed to fetch models: %w", err)
	}

	modelsList, ok := res.(external_models.OpenRouterModelsResponse)
	if !ok {
		c.Logger.Error("Invalid response format for models")
		return external_models.OpenRouterModelsResponse{}, fmt.Errorf("invalid response format for models")
	}

	modelsJSON, err := json.Marshal(modelsList)
	if err != nil {
		c.Logger.Error("Failed to marshal models for caching: ", err)
	} else {
		if err := rd.RedisSet(c.RedisClient, redisKey, string(modelsJSON), cacheDuration); err != nil {
			c.Logger.Error("Failed to cache models in Redis: ", err)
		} else {
			c.Logger.Info("Successfully cached models in Redis")
		}
	}

	return modelsList, nil
}

func (c *Client) ExtractModel(ctx *gin.Context, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest) (string, error) {
	var availableModels external_models.OpenRouterModelsResponse

	checkWebSearchModel := func(modelName string) bool {
		return strings.HasSuffix(modelName, ":online")
	}

	withTools := req.Tools != nil
	availableModels, _ = c.ListModels(extReq, withTools)

	models := availableModels.Data
	modelMap := make(map[string]bool)
	for _, model := range models {
		modelMap[model.ID] = true
	}

	var selectedModel string
	if headerModel := ctx.GetHeader("X-Model"); headerModel != "" {
		c.Logger.Info("Model selected via header: ", headerModel)
		selectedModel = strings.TrimSpace(headerModel)
	} else if queryModel := ctx.Query("model"); queryModel != "" {
		c.Logger.Info("Model selected via query: ", queryModel)
		selectedModel = strings.TrimSpace(queryModel)
	} else if req.Model != "" {
		c.Logger.Info("Model selected via body: ", req.Model)
		selectedModel = strings.TrimSpace(req.Model)
	} else {
		c.Logger.Info("Using default model")
		selectedModel = "deepseek/deepseek-r1-0528-qwen3-8b:free"
	}

	if !checkWebSearchModel(req.Model) {
		if _, exists := modelMap[selectedModel]; !exists {
			c.Logger.Error("Invalid model selected: ", selectedModel)
			return "deepseek/deepseek-r1-0528-qwen3-8b:free", fmt.Errorf("invalid model selected: %s", selectedModel)
		}

		return selectedModel, nil
	}

	selectedModel = strings.TrimSuffix(selectedModel, ":online")
	return strings.Join([]string{selectedModel, "online"}, ":"), nil
}

func ChargeAICreditUsage(db *storage.Database, ids models.IDS, inputLength int, logger *utility.Logger) error {
	var agentPrice float64 = 0.0

	creditUsed := models.CalculateCreditCost(inputLength, agentPrice)

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

	go models.PublishPlatformCreditUpdate(db.Postgresql)

	return nil
}

func ExtractChatContent(response map[string]interface{}) (string, error) {
	choices, ok := response["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", errors.New("invalid or empty choices")
	}

	firstChoice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", errors.New("invalid choice format")
	}

	message, ok := firstChoice["message"].(map[string]interface{})
	if !ok {
		return "", errors.New("invalid message format")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", errors.New("content not found or not a string")
	}

	cleaned := strings.TrimSpace(content)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	return cleaned, nil
}

func ConvertTools(src *[]models.Tool) *[]external_models.Tool {
	if src == nil {
		return nil
	}

	tools := make([]external_models.Tool, len(*src))
	for i, t := range *src {
		tools[i] = external_models.Tool{
			Type: t.Type,
			Function: external_models.Function{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters: external_models.ToolFunctionParameter{
					Type:       t.Function.Parameters.Type,
					Properties: t.Function.Parameters.Properties,
				},
			},
		}
	}

	return &tools
}
