package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

func (c *Client) ChatCompletions(db *storage.Database, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest, ids models.IDS) (map[string]any, int, error) {
	var telexlogs models.TelexAIUsageLog

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

	c.Logger.Info(fmt.Sprintf("Making request to model: %s for org: %s", req.GetModel(), ids.OrganisationID))

	res, err := extReq.SendExternalRequest(request.GetChatCompletions, openRouterPayload)
	if err != nil {
		if strings.Contains(err.Error(), "429") {
			c.Logger.Error("OpenRouter API call failed with 429: ", err)
			return map[string]any{}, http.StatusTooManyRequests, fmt.Errorf("rate limit exceeded: %w", err)
		}
		c.Logger.Error("OpenRouter API call failed: ", err)
		return map[string]any{}, http.StatusBadRequest, err
	}

	result, ok := res.(map[string]any)
	if !ok {
		c.Logger.Error("failed to get chat completions: ", res)
		return map[string]any{}, http.StatusBadRequest, fmt.Errorf("failed to get chat completions: %v", res)
	}

	if choices, exists := result["choices"].([]any); !exists || len(choices) == 0 {
		return map[string]any{}, http.StatusBadRequest, fmt.Errorf("no choices found in response")
	}

	if usage, exists := result["usage"].(external_models.OpenRouterUsage); exists {
		if err := telexlogs.CreateUsageLog(db.Postgresql, c.Logger, ids, req, usage); err != nil {
			c.Logger.Error("failed to create usage log: ", err)
			return map[string]any{}, http.StatusBadRequest, fmt.Errorf("failed to create usage log: %w", err)
		}
	}

	return result, http.StatusOK, nil
}

func (c *Client) StreamChatCompletions(w http.ResponseWriter, db *storage.Database, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest, ids models.IDS) error {
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

	c.Logger.Info(fmt.Sprintf("Starting stream for model: %s for org: %s", req.GetModel(), ids.OrganisationID))

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
			c.Logger.Error("Streaming error: ", chunk.Error)
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
		err = ChargeAICreditUsage(db, ids, fullContent.Len(), c.Logger)
		if err != nil {
			c.Logger.Error("Failed to charge credits after streaming", err)
		}
	}

	return nil
}

func (c *Client) StreamChatCompletionsChannel(ctx context.Context, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest) (<-chan external_models.StreamChunk, error) {
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

	c.Logger.Info(fmt.Sprintf("Starting stream for model: %s", req.GetModel()))

	streamChan, err := extReq.SendStreamingExternalRequest(request.GetChatCompletions, openRouterPayload, ctx)
	if err != nil {
		c.Logger.Error("Failed to start streaming: ", err)
		return nil, err
	}

	return streamChan, nil
}

func (c *Client) RespondToChat(w http.ResponseWriter, db *storage.Database, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest, ids models.IDS) (map[string]any, int, error) {
	response, code, err := c.ChatCompletions(db, req, extReq, ids)
	if err != nil {
		return map[string]any{}, code, err
	}

	content, err := ExtractChatContent(response)
	if err != nil {
		return map[string]any{}, http.StatusBadRequest, err
	}

	inputLength := len(content)
	err = ChargeAICreditUsage(db, ids, inputLength, c.Logger)
	if err != nil {
		return map[string]any{}, http.StatusBadRequest, err
	}

	return response, code, nil
}

func (c *Client) TranslatorCompletions(req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest) (map[string]any, int, error) {
	openRouterPayload := external_models.OpenRouterReq{
		Model:    "google/gemini-2.0-flash-001",
		Messages: req.Messages,
	}

	c.Logger.Info(fmt.Sprintf("Making request to model: %s for translator completions", req.GetModel()))
	res, err := extReq.SendExternalRequest(request.GetChatCompletions, openRouterPayload)
	if err != nil {
		if strings.Contains(err.Error(), "429") {
			c.Logger.Error("OpenRouter API call failed with 429: ", err)
			return map[string]any{}, http.StatusTooManyRequests, fmt.Errorf("rate limit exceeded: %w", err)
		}
		c.Logger.Error("OpenRouter API call failed: ", err)
		return map[string]any{}, http.StatusBadRequest, err
	}

	result, ok := res.(map[string]any)
	if !ok {
		c.Logger.Error("failed to get chat completions: ", res)
		return map[string]any{}, http.StatusBadRequest, fmt.Errorf("failed to get chat completions: %v", res)
	}

	if choices, exists := result["choices"].([]any); !exists || len(choices) == 0 {
		return map[string]any{}, http.StatusBadRequest, fmt.Errorf("no choices found in response")
	}

	return result, http.StatusOK, nil
}
