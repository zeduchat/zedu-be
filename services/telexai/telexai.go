package telexai

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func ChatCompletions(db *storage.Database, logger *utility.Logger, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest, ids models.IDS) (models.TelexAIChatCompletionsResp, int, error) {
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
	}

	logger.Info(fmt.Sprintf("Making request to model: %s for org: %s", req.GetModel(), ids.OrganisationID))
	res, err := extReq.SendExternalRequest(request.GetChatCompletions, openRouterPayload)
	if err != nil {
		if strings.Contains(err.Error(), "429") {
			logger.Error("OpenRouter API call failed with 429: ", err)
			return models.TelexAIChatCompletionsResp{}, http.StatusTooManyRequests, fmt.Errorf("rate limit exceeded: %w", err)
		}
		logger.Error("OpenRouter API call failed: ", err)
		return models.TelexAIChatCompletionsResp{}, http.StatusBadRequest, err
	}
	result, ok := res.(external_models.OpenRouterResp)
	if !ok {
		logger.Error("failed to get chat completions: ", err)
		return models.TelexAIChatCompletionsResp{}, http.StatusBadRequest, err
	}

	if len(result.Choices) == 0 {
		return models.TelexAIChatCompletionsResp{}, http.StatusBadRequest, fmt.Errorf("empty response")
	}

	if err := telexlogs.CreateUsageLog(db.Postgresql, logger, ids, req, result.Usage); err != nil {
		logger.Error("failed to create usage log: ", err)
		return models.TelexAIChatCompletionsResp{}, http.StatusBadRequest, fmt.Errorf("failed to create usage log: %w", err)
	}

	resp := models.TelexAIChatCompletionsResp{
		Messages: result.Choices[0].Message,
	}
	return resp, http.StatusOK, nil
}

func ListModels() (map[string]interface{}, error) {
	availableModels := map[string]interface{}{
		"models": []models.TelexAIModels{
			{
				ID:                   "deepseek/deepseek-r1-0528-qwen3-8b:free",
				Name:                 "DeepSeek R1 0528 Qwen3 8B",
				Provider:             "DeepSeek",
				Pricing:              "free",
				Context:              131072,
				CreatedAt:            "2025-05-29",
				InputCostPerMillion:  0,
				OutputCostPerMillion: 0,
				Type:                 "chat",
				Description:          "Distilled 8B-parameter version of DeepSeek-R1-0528-Qwen3. Rivals 235B models in reasoning and logic. Ideal for math and programming tasks.",
				Capabilities:         []string{"math", "reasoning", "code generation"},
				Tags:                 []string{"chain-of-thought", "distilled", "efficient"},
			},
			{
				ID:                   "deepseek/deepseek-r1-0528:free",
				Name:                 "DeepSeek R1 0528",
				Provider:             "DeepSeek",
				Pricing:              "free",
				Context:              163840,
				CreatedAt:            "2025-05-28",
				InputCostPerMillion:  0,
				OutputCostPerMillion: 0,
				Type:                 "chat",
				Description:          "Large-scale open-source model with 671B parameters. Delivers performance on par with GPT-4 and full reasoning traceability.",
				Capabilities:         []string{"reasoning", "long context", "open-source"},
				Tags:                 []string{"flagship", "transparent", "high-parameter"},
			},
			{
				ID:                   "qwen/qwen3-30b-a3b:free",
				Name:                 "Qwen3 30B A3B",
				Provider:             "Qwen",
				Pricing:              "free",
				Context:              40960,
				CreatedAt:            "2025-04-28",
				InputCostPerMillion:  0,
				OutputCostPerMillion: 0,
				Type:                 "chat",
				Description:          "MoE architecture with 30.5B parameters. Excels at multilingual, reasoning, coding, and dialogue. Strong balance of performance and efficiency.",
				Capabilities:         []string{"reasoning", "multilingual", "dialogue", "code generation"},
				Tags:                 []string{"moe", "creative", "math", "open-source"},
			},
			{
				ID:                   "meta-llama/llama-3.3-8b-instruct:free",
				Name:                 "Llama 3.3 8B Instruct",
				Provider:             "Meta",
				Pricing:              "free",
				Context:              128000,
				CreatedAt:            "2025-05-14",
				InputCostPerMillion:  0,
				OutputCostPerMillion: 0,
				Type:                 "chat",
				Description:          "Lightweight and fast variant of Llama 3.3 70B, optimized for low-latency instruction following.",
				Capabilities:         []string{"instruction-following", "low-latency", "general-purpose"},
				Tags:                 []string{"fast", "small-model", "open-source"},
			},
			{
				ID:                   "openai/gpt-3.5-turbo",
				Name:                 "GPT-3.5 Turbo",
				Provider:             "OpenAI",
				Pricing:              "standard",
				Context:              16385,
				CreatedAt:            "2023-05-28",
				InputCostPerMillion:  0.5,
				OutputCostPerMillion: 1.5,
				Type:                 "chat",
				Description:          "OpenAI’s fastest model for general-purpose chat and code generation. Optimized for cost and speed.",
				Capabilities:         []string{"chat", "code", "completion"},
				Tags:                 []string{"cost-efficient", "fast", "reliable"},
			},
			{
				ID:                   "openai/gpt-4",
				Name:                 "GPT-4",
				Provider:             "OpenAI",
				Pricing:              "premium",
				Context:              8191,
				CreatedAt:            "2023-05-28",
				InputCostPerMillion:  30,
				OutputCostPerMillion: 60,
				Type:                 "chat",
				Description:          "Flagship multimodal model with superior reasoning and general knowledge. Best for complex tasks and professional use.",
				Capabilities:         []string{"multimodal", "advanced reasoning", "long-form generation"},
				Tags:                 []string{"premium", "high-accuracy", "multimodal"},
			},
		},
	}
	return availableModels, nil
}

func ExtractModel(c *gin.Context, logger *utility.Logger, req models.TelexAIChatCompletionsReq) string {
	availableModels, _ := ListModels()
	models := availableModels["models"].([]models.TelexAIModels)
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
		return "deepseek/deepseek-r1-0528-qwen3-8b:free"
	}
	return selectedModel
}
