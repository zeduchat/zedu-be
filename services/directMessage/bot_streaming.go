package dm

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/openrouter"
	"github.com/hngprojects/telex_be/utility"
)

func ProcessBotStreamingResponse(req models.BotRequest, orgAgent models.OrganisationIntegrations, channel models.DmChannels, extReq request.ExternalRequest, logger *utility.Logger) (string, error) {
	req.BotNotification = models.AgentProcessingStarted
	err := SendAgentNotification(req, logger)
	if err != nil {
		return "", fmt.Errorf("failed to send agent processing started: %v", err)
	}

	client := openrouter.GetClient()
	if client == nil {
		logger.Error("OpenRouter client not initialized")
		return "", fmt.Errorf("OpenRouter client not initialized")
	}

	tools := InitializeTools()

	// Strip HTML tags
	re := regexp.MustCompile("<[^>]*>")
	strippedContent := re.ReplaceAllString(req.Content, "")

	var messages []external_models.TelexAIOpenRouterMessage

	// Add system message
	messages = append(messages, external_models.TelexAIOpenRouterMessage{
		Role:    "system",
		Content: fmt.Sprintf("You are %s. %s", orgAgent.AppName, orgAgent.AppDescription),
	})

	// Prepare user content
	var userContent any
	if len(req.Media) > 0 {
		var contentParts []map[string]any
		if strippedContent != "" {
			contentParts = append(contentParts, map[string]any{
				"type": "text",
				"text": strippedContent,
			})
		}
		for _, media := range req.Media {
			contentParts = append(contentParts, map[string]any{
				"type": "image_url",
				"image_url": map[string]string{
					"url": media.FileLink,
				},
			})
		}
		userContent = contentParts
	} else {
		userContent = strippedContent
	}

	messages = append(messages, external_models.TelexAIOpenRouterMessage{
		Role:    "user",
		Content: userContent,
	})

	chatReq := models.TelexAIChatCompletionsReq{
		Model:    "google/gemini-2.0-flash-exp:free",
		Messages: messages,
		Tools:    &tools,
	}

	ctx := context.Background()
	streamChan, err := client.StreamChatCompletionsChannel(ctx, chatReq, extReq)
	if err != nil {
		req.BotNotification = models.AgentErrorOccured
		SendAgentNotification(req, logger)
		return "", fmt.Errorf("failed to start streaming: %v", err)
	}

	var fullContent strings.Builder
	var toolCalls []external_models.ToolCall
	processingFinishedSent := false

	for chunk := range streamChan {
		if chunk.Error != nil {
			logger.Error("Streaming error: ", chunk.Error)
			req.BotNotification = models.AgentErrorOccured
			SendAgentNotification(req, logger)
			return "", chunk.Error
		}

		if chunk.Done {
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
						break
					}

					if data != "" && data != "[DONE]" {
						var parsed map[string]any
						if err := json.Unmarshal([]byte(data), &parsed); err == nil {
							if choices, ok := parsed["choices"].([]any); ok && len(choices) > 0 {
								if choice, ok := choices[0].(map[string]any); ok {
									if delta, ok := choice["delta"].(map[string]any); ok {
										if content, ok := delta["content"].(string); ok {
											if !processingFinishedSent {
												req.BotNotification = models.AgentProcessingFinished
												SendAgentNotification(req, logger)
												processingFinishedSent = true
											}

											fullContent.WriteString(content)

											feed := models.FeedMessageRequest{
												ChannelID: req.ChannelID,
												UserName:  orgAgent.AppName,
												CreatedAt: time.Now().UTC().Format(time.RFC3339),
												AvatarURL: orgAgent.AppLogo,
												Type:      "message",
												Content:   fullContent.String(),
												ThreadId:  req.ThreadId,
												Email:     "agent",
												FullName:  orgAgent.AppName,
												UserId:    *channel.ParticipantId,
												Media:     req.Media,
												UserType:  "bot",
												State:     req.State,
											}

											notification := models.Notification[models.AgentUpdate]
											notification.SectionType = models.ThreadSection
											notification.Content = feed
											notification.ModificationDetails = &models.ModificationDetails{
												ThreadId:  req.ThreadId,
												ChannelId: req.ChannelID,
											}

											err = centrifuge.PublishChannel(logger, req.ChannelID, notification)
											logger.Info("Published chunk to channel [%s] for agent: [%s]", req.ChannelID, req.AgentId)
											if err != nil {
												logger.Error(fmt.Sprintf("Error Publishing chunk to channel %s: %v", req.ChannelID, err.Error()))
											}
										}

										if toolCallsData, ok := delta["tool_calls"].([]any); ok {
											for _, tc := range toolCallsData {
												if toolCallMap, ok := tc.(map[string]any); ok {
													var toolCall external_models.ToolCall
													tcBytes, _ := json.Marshal(toolCallMap)
													json.Unmarshal(tcBytes, &toolCall)
													toolCalls = append(toolCalls, toolCall)
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
	}

	if len(toolCalls) > 0 {
		logger.Info(fmt.Sprintf("Processing %d tool calls", len(toolCalls)))

		toolResults, err := ExecuteToolCalls(toolCalls, logger, req)
		if err != nil {
			logger.Error(fmt.Sprintf("Error executing tool calls: %v", err))
			return fullContent.String(), nil
		}

		chatReq.Messages = append(chatReq.Messages, external_models.TelexAIOpenRouterMessage{
			Role:      "assistant",
			Content:   fullContent.String(),
			ToolCalls: toolCalls,
		})

		for _, result := range toolResults {
			chatReq.Messages = append(chatReq.Messages, result)
		}

		streamChan, err = client.StreamChatCompletionsChannel(ctx, chatReq, extReq)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to continue conversation after tool calls: %v", err))
			return fullContent.String(), nil
		}

		fullContent.Reset()

		for chunk := range streamChan {
			if chunk.Error != nil {
				logger.Error("Streaming error after tool call: ", chunk.Error)
				break
			}

			if chunk.Done {
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
							break
						}

						if data != "" && data != "[DONE]" {
							var parsed map[string]any
							if err := json.Unmarshal([]byte(data), &parsed); err == nil {
								if choices, ok := parsed["choices"].([]any); ok && len(choices) > 0 {
									if choice, ok := choices[0].(map[string]any); ok {
										if delta, ok := choice["delta"].(map[string]any); ok {
											if content, ok := delta["content"].(string); ok {
												fullContent.WriteString(content)

												feed := models.FeedMessageRequest{
													ChannelID: req.ChannelID,
													UserName:  orgAgent.AppName,
													CreatedAt: time.Now().UTC().Format(time.RFC3339),
													AvatarURL: orgAgent.AppLogo,
													Type:      "message",
													Content:   fullContent.String(),
													ThreadId:  req.ThreadId,
													Email:     "agent",
													FullName:  orgAgent.AppName,
													UserId:    *channel.ParticipantId,
													Media:     req.Media,
													UserType:  "bot",
													State:     req.State,
												}

												notification := models.Notification[models.AgentUpdate]
												notification.SectionType = models.ThreadSection
												notification.Content = feed
												notification.ModificationDetails = &models.ModificationDetails{
													ThreadId:  req.ThreadId,
													ChannelId: req.ChannelID,
												}

												err = centrifuge.PublishChannel(logger, req.ChannelID, notification)
												if err != nil {
													logger.Error(fmt.Sprintf("Error Publishing chunk to channel %s: %v", req.ChannelID, err.Error()))
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
	}

	return fullContent.String(), nil
}

func SendAgentNotification(req models.BotRequest, logger *utility.Logger) error {
	notification := models.Notification[req.BotNotification]
	notification.SectionType = models.ThreadSection
	notification.ModificationDetails = &models.ModificationDetails{
		ThreadId:  req.ThreadId,
		ChannelId: req.ChannelID,
	}

	err := centrifuge.PublishChannel(logger, req.ChannelID, notification)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing notification to channel %s: %v", req.ChannelID, err.Error()))
	}

	logger.Info("Published bot notification: [%s] to channel [%s]", req.BotNotification, req.ChannelID)

	return nil
}
