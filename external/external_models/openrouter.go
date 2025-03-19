package external_models

type OpenRouterReq struct {
	Model    string                     `json:"model" validate:"required"`
	Messages []TelexAIOpenRouterMessage `json:"messages" validate:"required"`
}

type OpenRouterResp struct {
	ID      string             `json:"id" validate:"required"`
	Choices []OpenRouterChoice `json:"choices" validate:"required"`
	Usage   OpenRouterUsage    `json:"usage"`
}

type OpenRouterChoice struct {
	Message TelexAIOpenRouterMessage `json:"message" validate:"required"`
}

type OpenRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type TelexAIOpenRouterMessage struct {
	Role    string `json:"role" validate:"required,oneof=system developer user assistant tool"`
	Content string `json:"content" validate:"required"`
}
