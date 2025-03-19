package models

import "github.com/hngprojects/telex_be/external/external_models"

// AIProxyChatRequest holds fields for making
// post request to the chat completions endpoint
type TelexAIChatCompletionsReq struct {
	Messages []external_models.TelexAIOpenRouterMessage `json:"messages" validate:"required"`
	OrgID    string                                     `json:"org_id" validate:"required"`
}

// AIProxyChatCompletionsResp holds fields for response for openRouter requests
type TelexAIChatCompletionsResp struct {
	Messages external_models.TelexAIOpenRouterMessage
}
