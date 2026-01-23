package dm

import (
	"encoding/json"
	"fmt"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/openrouter/tools"
	"github.com/hngprojects/telex_be/pkg/repository/openrouter/tools/capitalize"
	"github.com/hngprojects/telex_be/utility"
)

var toolRegistry map[string]tools.ToolExecutor

func InitializeTools() []models.Tool {
	toolRegistry = make(map[string]tools.ToolExecutor)

	capitalizeTool := capitalize.NewCapitalizeTool()
	toolRegistry["capitalize_text"] = capitalizeTool

	toolDefinitions := []models.Tool{}

	for _, tool := range toolRegistry {
		def := tool.GetDefinition()
		toolDefinitions = append(toolDefinitions, models.Tool{
			Type: def.Type,
			Function: models.ToolFunction{
				Name:        def.Function.Name,
				Description: def.Function.Description,
				Parameters: models.ToolFunctionParameter{
					Type:       def.Function.Parameters["type"].(string),
					Properties: def.Function.Parameters["properties"].(map[string]any),
					Required:   def.Function.Parameters["required"].([]string),
				},
			},
		})
	}

	return toolDefinitions
}

func ExecuteToolCalls(toolCalls []external_models.ToolCall, logger *utility.Logger, req models.BotRequest) ([]external_models.TelexAIOpenRouterMessage, error) {
	var results []external_models.TelexAIOpenRouterMessage

	for _, toolCall := range toolCalls {
		if toolCall.Function == nil {
			continue
		}

		toolName := toolCall.Function.Name
		executor, exists := toolRegistry[toolName]
		if !exists {
			logger.Error(fmt.Sprintf("Tool not found: %s", toolName))
			continue
		}

		var arguments map[string]interface{}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments); err != nil {
			logger.Error(fmt.Sprintf("Failed to parse tool arguments: %v", err))
			continue
		}

		toolStartedData := models.ToolCallNotification{
			ToolName:  toolName,
			Arguments: arguments,
			Status:    "started",
			Result:    nil,
			Error:     nil,
		}

		req.BotNotification = models.AgentToolCallStarted
		SendToolCallNotification(req, toolStartedData, logger)

		result, err := executor.Execute(arguments)
		if err != nil {
			logger.Error(fmt.Sprintf("Tool execution failed for %s: %v", toolName, err))

			errorMsg := err.Error()
			toolErrorData := models.ToolCallNotification{
				ToolName:  toolName,
				Arguments: arguments,
				Status:    "error",
				Result:    nil,
				Error:     &errorMsg,
			}
			req.BotNotification = models.AgentErrorOccured
			SendToolCallNotification(req, toolErrorData, logger)
			continue
		}

		resultJSON, err := json.Marshal(result)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to marshal tool result: %v", err))
			continue
		}

		toolCompletedData := models.ToolCallNotification{
			ToolName:  toolName,
			Arguments: arguments,
			Status:    "completed",
			Result:    result,
			Error:     nil,
		}

		req.BotNotification = models.AgentToolCallCompleted
		SendToolCallNotification(req, toolCompletedData, logger)

		results = append(results, external_models.TelexAIOpenRouterMessage{
			Role:       "tool",
			Content:    string(resultJSON),
			ToolCallID: toolCall.ID,
			Name:       toolName,
		})

		logger.Info(fmt.Sprintf("Executed tool %s successfully", toolName))
	}

	return results, nil
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
