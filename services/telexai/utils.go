package telexai

import (
	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/internal/models"
)

func ConvertTools(src *models.Tools) *external_models.Tools {
	if src == nil {
		return nil
	}

	tools := make([]external_models.Tool, len(src.Tools))
	for i, t := range src.Tools{
		tools[i] = external_models.Tool{
			Type: t.Type,
			Function: external_models.Function{
				Name: t.Function.Name,
				Description: t.Function.Description,
				Parameters: external_models.ToolFunctionParameter{
					Type: t.Function.Parameters.Type,
					Properties: t.Function.Parameters.Properties,
					Required: t.Function.Parameters.Required,
				},
			},
		}
	}

	return &external_models.Tools{Tools: tools}
}
