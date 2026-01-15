package telexai

import (
	"context"
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

func RespondToChat(w http.ResponseWriter, db *storage.Database, logger *utility.Logger, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest, ids models.IDS) (map[string]any, int, error) {

	response, code, err := ChatCompletions(w, db, logger, req, extReq, ids)
	if err != nil {
		return map[string]any{}, code, err
	}

	content, err := ExtractChatContent(response)
	if err != nil {
		return map[string]any{}, http.StatusBadRequest, err
	}

	inputLength := len(content)
	err = ChargeAICreditUsage(db, ids, inputLength, logger)
	if err != nil {
		return map[string]any{}, http.StatusBadRequest, err
	}

	return response, code, nil
}

func StreamChatCompletions(w http.ResponseWriter, db *storage.Database, logger *utility.Logger, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest, ids models.IDS) error {

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	openRouterPayload := external_models.OpenRouterReq{
		Model:    req.GetModel(),
		Messages: req.Messages,
		ExtraBody: external_models.OpenRouterExtraBody{
			Usage: external_models.OpenRouterUsageToggle{
				Include: true,
			},
		},
		Tools:  ConvertTools(req.Tools),
		Stream: true,
	}

	logger.Info(fmt.Sprintf("Starting stream for model: %s for org: %s", req.GetModel(), ids.OrganisationID))

	ctx := context.Background()

	streamChan, err := extReq.SendStreamingExternalRequest(request.GetChatCompletions, openRouterPayload, ctx)
	if err != nil {
		fmt.Fprintf(w, "data: {\"error\": \"%s\"}\n\n", err.Error())
		flusher.Flush()
		return err
	}

	var fullContent strings.Builder

	for chunk := range streamChan {
		if chunk.Error != nil {
			logger.Error("Streaming error: ", chunk.Error)
			fmt.Fprintf(w, "data: {\"error\": \"%s\"}\n\n", chunk.Error.Error())
			flusher.Flush()
			return chunk.Error
		}

		if chunk.Done {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			break
		}

		if len(chunk.Data) > 0 {
			dataStr := string(chunk.Data)
			lines := strings.Split(dataStr, "\n")

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "data: ") {
					data := line[6:]
					if data == "[DONE]" {
						fmt.Fprintf(w, "data: [DONE]\n\n")
						flusher.Flush()
						goto billing
					}

					if data != "" && data != "[DONE]" {
						fmt.Fprintf(w, "data: %s\n\n", data)
						flusher.Flush()

						// Extract content for billing (optional)
						if data != "" {
							var parsed map[string]any
							if err := json.Unmarshal([]byte(data), &parsed); err == nil {
								if choices, ok := parsed["choices"].([]any); ok && len(choices) > 0 {
									if choice, ok := choices[0].(map[string]any); ok {
										if delta, ok := choice["delta"].(map[string]any); ok {
											if content, ok := delta["content"].(string); ok {
												fullContent.WriteString(content)
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

billing:
	if fullContent.Len() > 0 {
		err = ChargeAICreditUsage(db, ids, fullContent.Len(), logger)
		if err != nil {
			logger.Error("Failed to charge credits after streaming", err)
		}
	}

	return nil
}

func ChatCompletions(w http.ResponseWriter, db *storage.Database, logger *utility.Logger, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest, ids models.IDS) (map[string]any, int, error) {
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
		Tools:  ConvertTools(req.Tools),
		Stream: false,
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

func TranslatorCompletions(logger *utility.Logger, extReq request.ExternalRequest, req models.TelexAIChatCompletionsReq) (map[string]any, int, error) {
	openRouterPayload := external_models.OpenRouterReq{
		Model: "google/gemini-2.0-flash-001",
		// Model: "anthropic/claude-3.5-haiku",
		// Model: "openai/gpt-4.1-nano",
		Messages: req.Messages,
	}

	logger.Info(fmt.Sprintf("Making request to model: %s for translator completions", req.GetModel()))
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

	return result, http.StatusOK, nil
}

func ExtractModel(c *gin.Context, logger *utility.Logger, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest, redis *redis.Client) (string, error) {
	var (
		availableModels external_models.OpenRouterModelsResponse
	)

	checkWebSearchModel := func(modelName string) bool {
		return strings.HasSuffix(modelName, ":online")
	}

	withTools := req.Tools != nil
	availableModels, _ = ListModels(logger, extReq, redis, withTools)

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

	if !checkWebSearchModel(req.Model) {
		if _, exists := modelMap[selectedModel]; !exists {
			logger.Error("Invalid model selected: ", selectedModel)
			return "deepseek/deepseek-r1-0528-qwen3-8b:free", fmt.Errorf("invalid model selected: %s", selectedModel)
		}

		return selectedModel, nil
	}

	selectedModel = strings.TrimSuffix(selectedModel, ":online")
	return strings.Join([]string{selectedModel, "online"}, ":"), nil
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

	// Publish real-time update to superadmin dashboard (async)
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
