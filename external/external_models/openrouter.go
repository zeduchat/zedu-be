package external_models

type OpenRouterReq struct {
	Model     string                     `json:"model" validate:"required"`
	Messages  []TelexAIOpenRouterMessage `json:"messages" validate:"required"`
	ExtraBody OpenRouterExtraBody        `json:"extra_body,omitempty"`
	Stream    bool                       `json:"stream" validate:"required,oneof=true false" default:"false"` // Whether to stream the response or not
}

type OpenRouterExtraBody struct {
	Usage OpenRouterUsageToggle `json:"usage"`
}
type OpenRouterUsageToggle struct {
	Include bool `json:"include"`
}

type OpenRouterChoice struct {
	Message TelexAIOpenRouterMessage `json:"message" validate:"required"`
}

type OpenRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	Cost             int `json:"cost"` 
}

type TelexAIOpenRouterMessage struct {
	Role    string `json:"role" validate:"required,oneof=system developer user assistant tool"`
	Content string `json:"content" validate:"required"`
}

type OpenRouterResp struct {
	ID      string             `json:"id"`
	Choices []OpenRouterChoice `json:"choices"`
	Usage   OpenRouterUsage    `json:"usage"`
}
