package external_models

type OpenRouterReq struct {
	Model     string                     `json:"model" validate:"required"`
	Messages  []TelexAIOpenRouterMessage `json:"messages" validate:"required"`
	ExtraBody OpenRouterExtraBody        `json:"extra_body,omitempty"`
	Stream    bool                       `json:"stream" validate:"required,oneof=true false" default:"false"` // Whether to stream the response or not
	Tools     *[]Tool                    `json:"tools,omitempty" validate:"omitempty,dive"`
}

type OpenRouterExtraBody struct {
	Usage OpenRouterUsageToggle `json:"usage"`
}
type OpenRouterUsageToggle struct {
	Include bool `json:"include"`
}

type TelexAIOpenRouterMessage struct {
	Role       string     `json:"role" validate:"required,oneof=system developer user assistant tool"`
	Content    any        `json:"content" validate:"required"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type ToolFunctionParameter struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

type Function struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Parameters  ToolFunctionParameter `json:"parameters"`
}

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type ToolCall struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Function *ToolFunction `json:"function,omitempty"`
}

type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type OpenRouterResp struct {
	ID      string             `json:"id"`
	Choices []OpenRouterChoice `json:"choices"`
	Model   string             `json:"model,omitempty"`
	Usage   OpenRouterUsage    `json:"usage"`
}

type OpenRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	Cost             int `json:"cost"`
}
type OpenRouterChoice struct {
	Message            TelexAIOpenRouterMessage `json:"message" validate:"required"`
	FinishReason       string                   `json:"finish_reason,omitempty"`
	NativeFinishReason string                   `json:"native_finish_reason,omitempty"`
}

type OpenRouterModelsResponse struct {
	Data []AllOpenRouterModelsResp `json:"data"`
}

type AllOpenRouterModelsResp struct {
	ID                  string       `json:"id"`
	HuggingFaceID       string       `json:"hugging_face_id"`
	Name                string       `json:"name"`
	Created             int64        `json:"created"`
	Description         string       `json:"description"`
	ContextLength       int          `json:"context_length"`
	Architecture        Architecture `json:"architecture"`
	Pricing             Pricing      `json:"pricing"`
	TopProvider         TopProvider  `json:"top_provider"`
	PerRequestLimits    *string      `json:"per_request_limits"`
	SupportedParameters []string     `json:"supported_parameters"`
}

type Architecture struct {
	Modality         string   `json:"modality"`
	InputModalities  []string `json:"input_modalities"`
	OutputModalities []string `json:"output_modalities"`
	Tokenizer        string   `json:"tokenizer"`
	InstructType     *string  `json:"instruct_type"`
}

type Pricing struct {
	Prompt            string `json:"prompt"`
	Completion        string `json:"completion"`
	Request           string `json:"request"`
	Image             string `json:"image"`
	WebSearch         string `json:"web_search"`
	InternalReasoning string `json:"internal_reasoning"`
}

type TopProvider struct {
	ContextLength       int  `json:"context_length"`
	MaxCompletionTokens *int `json:"max_completion_tokens"`
	IsModerated         bool `json:"is_moderated"`
}
