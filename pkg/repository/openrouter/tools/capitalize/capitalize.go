package capitalize

import (
	"fmt"
	"strings"

	"github.com/hngprojects/telex_be/pkg/repository/openrouter/tools"
)

type CapitalizeTool struct{}

func (t *CapitalizeTool) Execute(arguments map[string]interface{}) (interface{}, error) {
	text, ok := arguments["text"].(string)
	if !ok {
		return nil, fmt.Errorf("text parameter is required and must be a string")
	}

	capitalized := strings.ToUpper(text)

	return map[string]interface{}{
		"original":    text,
		"capitalized": capitalized,
	}, nil
}

func (t *CapitalizeTool) GetDefinition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Type: "function",
		Function: tools.Function{
			Name:        "capitalize_text",
			Description: "Capitalizes all characters in the provided text to uppercase",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "The text to capitalize",
					},
				},
				"required": []string{"text"},
			},
		},
	}
}

func NewCapitalizeTool() *CapitalizeTool {
	return &CapitalizeTool{}
}
