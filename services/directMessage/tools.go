package dm

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	agentruntime "github.com/hngprojects/telex_be/services/agents"
	"github.com/hngprojects/telex_be/utility"
)

type ToolExecutionRequest struct {
	DB         *gorm.DB
	BotRequest models.BotRequest
	Logger     *utility.Logger
}

func InitializeTools() []models.Tool {
	return agentruntime.ToolDefinitions()
}

func ExecuteToolCalls(toolCalls []external_models.ToolCall, input ToolExecutionRequest) ([]external_models.TelexAIOpenRouterMessage, error) {
	if input.Logger == nil {
		return nil, fmt.Errorf("logger is not initialized")
	}

	registry := agentruntime.NewToolRegistry()
	context := agentruntime.ToolExecutionContext{
		DB:        input.DB,
		AgentID:   input.BotRequest.AgentId,
		OrgID:     input.BotRequest.OrgId,
		UserID:    input.BotRequest.UserId,
		ChannelID: input.BotRequest.ChannelID,
		ThreadID:  input.BotRequest.ThreadId,
	}

	results := make([]external_models.TelexAIOpenRouterMessage, 0, len(toolCalls))

	for _, toolCall := range toolCalls {
		if toolCall.Function == nil {
			continue
		}

		toolName := toolCall.Function.Name
		var arguments map[string]any
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments); err != nil {
			input.Logger.Error(fmt.Sprintf("Failed to parse tool arguments for %s: %v", toolName, err))
			results = append(results, toolErrorMessage(toolCall.ID, toolName, fmt.Errorf("invalid tool arguments: %w", err)))
			continue
		}

		toolStartedData := models.ToolCallNotification{
			ToolName:  toolName,
			Arguments: arguments,
			Status:    "started",
			Result:    nil,
			Error:     nil,
		}
		input.BotRequest.BotNotification = models.AgentToolCallStarted
		SendToolCallNotification(input.BotRequest, toolStartedData, input.Logger)

		result, err := registry.Execute(context, toolName, arguments)
		if err != nil {
			input.Logger.Error(fmt.Sprintf("Tool execution failed for %s: %v", toolName, err))
			errorMsg := err.Error()
			toolErrorData := models.ToolCallNotification{
				ToolName:  toolName,
				Arguments: arguments,
				Status:    "error",
				Result:    nil,
				Error:     &errorMsg,
			}
			input.BotRequest.BotNotification = models.AgentErrorOccured
			SendToolCallNotification(input.BotRequest, toolErrorData, input.Logger)
			results = append(results, toolErrorMessage(toolCall.ID, toolName, err))
			continue
		}

		resultJSON, err := json.Marshal(result)
		if err != nil {
			input.Logger.Error(fmt.Sprintf("Failed to marshal tool result for %s: %v", toolName, err))
			results = append(results, toolErrorMessage(toolCall.ID, toolName, fmt.Errorf("failed to serialize tool result: %w", err)))
			continue
		}

		toolCompletedData := models.ToolCallNotification{
			ToolName:  toolName,
			Arguments: arguments,
			Status:    "completed",
			Result:    result,
			Error:     nil,
		}
		input.BotRequest.BotNotification = models.AgentToolCallCompleted
		SendToolCallNotification(input.BotRequest, toolCompletedData, input.Logger)

		results = append(results, external_models.TelexAIOpenRouterMessage{
			Role:       "tool",
			Content:    string(resultJSON),
			ToolCallID: toolCall.ID,
			Name:       toolName,
		})

		input.Logger.Info(fmt.Sprintf("Executed tool %s successfully", toolName))
	}

	return results, nil
}

func toolErrorMessage(toolCallID, toolName string, err error) external_models.TelexAIOpenRouterMessage {
	payload := map[string]string{"error": err.Error()}
	content, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		content = []byte(fmt.Sprintf("{\"error\":%q}", err.Error()))
	}

	return external_models.TelexAIOpenRouterMessage{
		Role:       "tool",
		Content:    string(content),
		ToolCallID: toolCallID,
		Name:       toolName,
	}
}

func SendToolCallNotification(req models.BotRequest, toolData models.ToolCallNotification, logger *utility.Logger) error {
	notification := models.Notification[req.BotNotification]
	notification.SectionType = models.ThreadSection
	notification.Content = toolData
	notification.ModificationDetails = &models.ModificationDetails{
		ThreadId:  req.ThreadId,
		ChannelId: req.ChannelID,
	}

	err := centrifuge.PublishChannel(logger, req.ChannelID, notification)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing tool notification to channel %s: %v", req.ChannelID, err.Error()))
	}

	logger.Info("Published tool call notification: [%s] for tool [%s] to channel [%s]", req.BotNotification, toolData.ToolName, req.ChannelID)

	return nil
}
