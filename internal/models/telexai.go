package models

import (
	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

// AIProxyChatRequest holds fields for making
// post request to the chat completions endpoint
type TelexAIChatCompletionsReq struct {
	Messages       []external_models.TelexAIOpenRouterMessage `json:"messages" validate:"required"`
	Model          string                                     `json:"model"` // Model to use for the chat completion
	Stream         bool                                       `json:"stream"`                    // Whether to stream the response or not
	// MaxTokens   *int                       `json:"max_tokens,omitempty"`
	// Temperature *float64                   `json:"temperature,omitempty"`
}

// AIProxyChatCompletionsResp holds fields for response for openRouter requests
type TelexAIChatCompletionsResp struct {
	Messages external_models.TelexAIOpenRouterMessage
}

func (treq *TelexAIChatCompletionsReq) GetModel() string {
	if treq.Model == "" {
		return "google/gemma-3-4b-it:free"
	}
	return treq.Model
}

func (treq *TelexAIChatCompletionsReq) IsValidModel() bool {
	validModels := map[string]bool{
		"openai/gpt-4":                            true,
		"openai/gpt-3.5-turbo":                    true,
		"meta-llama/llama-3.3-8b-instruct:free":   true,
		"deepseek/deepseek-r1-0528-qwen3-8b:free": true,
		"deepseek/deepseek-r1-0528:free":          true,
		"qwen/qwen3-30b-a3b:free":                 true,
		// more models to be added later on
	}

	return validModels[treq.GetModel()]
}

type TelexAIUsageLog struct {
	ID               string `gorm:"type:uuid;primary_key" json:"id"`
	OrganisationID   string `gorm:"type:uuid;not null" json:"organisation_id"`
	AgentID          string `gorm:"type:uuid;not null" json:"agent_id"`
	Model            string `gorm:"type:varchar(255);not null" json:"model"`
	PromptTokens     int    `gorm:"type:int;not null" json:"prompt_tokens"`
	CompletionTokens int    `gorm:"type:int;not null" json:"completion_tokens"`
	TotalTokens      int    `gorm:"type:int;not null" json:"total_tokens"`
	Cost             int    `gorm:"type:int;not null" json:"cost"`
	CreatedAt        int64  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        int64  `gorm:"autoUpdateTime" json:"updated_at"`
}

  type TelexAIModels struct {
    ID                   string   `json:"id"`
    Name                 string   `json:"name"`
    Provider             string   `json:"provider"`
    Pricing              string   `json:"pricing"`
    Context              int      `json:"context"`
    CreatedAt            string   `json:"created_at"`
    InputCostPerMillion  float64  `json:"input_cost_per_million"`
    OutputCostPerMillion float64  `json:"output_cost_per_million"`
    Type                 string   `json:"type"`
    Description          string   `json:"description"`
    Capabilities         []string `json:"capabilities"`
    Tags                 []string `json:"tags"`
    URL                  string   `json:"url"`
  }

func (t *TelexAIUsageLog) CreateUsageLog(db *gorm.DB, logger *utility.Logger, ids IDS, req TelexAIChatCompletionsReq, usage external_models.OpenRouterUsage) error {
	t.ID = utility.GenerateUUID()
	t.OrganisationID = ids.OrganisationID
	t.AgentID = ids.AgentID
	t.Model = req.GetModel()
	t.PromptTokens = usage.PromptTokens
	t.CompletionTokens = usage.CompletionTokens
	t.TotalTokens = usage.TotalTokens
	t.Cost = usage.Cost

	if err := db.Create(t).Error; err != nil {
		logger.Error("failed to create usage log: ", err)
		return err
	}
	return nil
}
