package models

import (
	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

// AIProxyChatRequest holds fields for making
// post request to the chat completions endpoint
type TelexAIChatCompletionsReq struct {
	Messages         []external_models.TelexAIOpenRouterMessage `json:"messages" validate:"required"`
	Model            string                                     `json:"model"`
	Stream           bool                                       `json:"stream"`
	Tools            *[]Tool                                    `json:"tools"`
	Plugins          []WebPlugin                                `json:"plugins,omitempty"`
	WebSearchOptions WebSearchOptions                           `json:"web_search_options,omitempty"`
	// MaxTokens   *int                       `json:"max_tokens,omitempty"`
	// Temperature *float64                   `json:"temperature,omitempty"`
}

type WebPlugin struct {
	ID           string `json:"id"`
	MaxResults   int    `json:"max_results,omitempty"`
	SearchPrompt string `json:"search_prompt,omitempty"`
}

type WebSearchOptions struct {
	SearchContextSize string `json:"search_context_size"`
}

type ToolFunctionParameter struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	Required   []string       `json:"required"`
}

type ToolFunction struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Parameters  ToolFunctionParameter `json:"parameters"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

func (treq *TelexAIChatCompletionsReq) GetModel() string {
	if treq.Model == "" {
		return "openai/gpt-4o"
	}
	return treq.Model
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
	// URL                  string   `json:"url"`
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
