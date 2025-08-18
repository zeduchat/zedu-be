package translator

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/telexai"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func GenerateTranslation(db *gorm.DB, logger *utility.Logger, extReq request.ExternalRequest, req models.TranslationRequest) (models.TranslationResponse, int, error) {

	stepProcess, err := runTranslationPipeline(db, logger, extReq, req.TaskList, req)
	if err != nil {
		return models.TranslationResponse{}, http.StatusBadRequest, err
	}

	resp := models.TranslationResponse{
		Status:      "success",
		ProcessStep: stepProcess,
	}

	return resp, http.StatusOK, nil
}

func runTranslationPipeline(db *gorm.DB, logger *utility.Logger, extReq request.ExternalRequest, tasklist string, req models.TranslationRequest) ([]models.ProcessStep, error) {
	stepProcess := []models.ProcessStep{}
	placeholders := map[string]string{
		"agent_skills":  strings.Join(req.AgentSkills, ", "),
		"global_skills": strings.Join(req.GlobalSkills, ", "),
	}

	var previousOutput string = tasklist

	for _, step := range req.Steps {
		var prompt models.Prompts

		if _, err := prompt.GetPromptByVersion(db, step); err != nil {
			return stepProcess, err
		}

		pStep := models.ProcessStep{
			Step:    step.Name,
			Input:   previousOutput,
			Status:  "in_progress",
			LLMCall: true,
			Prompt:  prompt.Template,
		}

		if pStep.LLMCall {
			systemPrompt := assemblePrompt(pStep, placeholders)
			aiOutput, _, err := LLMCall(logger, extReq, systemPrompt, pStep.Input)
			if err != nil {
				return stepProcess, err
			}
			pStep.Output = aiOutput

		} else {
			pStep.Output = pStep.Input
		}

		previousOutput = pStep.Output

		pStep.Status = "completed"
		stepProcess = append(stepProcess, pStep)
	}

	return stepProcess, nil
}

func assemblePrompt(step models.ProcessStep, placeholderValues map[string]string) string {
	stepPrompt := step.Prompt

	for placeholder, value := range placeholderValues {
		target := fmt.Sprintf("{{%s}}", placeholder)
		if strings.Contains(stepPrompt, target) {
			stepPrompt = strings.ReplaceAll(stepPrompt, target, value)
		}
	}

	return stepPrompt
}

func LLMCall(logger *utility.Logger, extReq request.ExternalRequest, systemPrompt, stepInput string) (string, int, error) {

	req := models.TelexAIChatCompletionsReq{
		Messages: []external_models.TelexAIOpenRouterMessage{
			{
				Role:    "system",
				Content: &systemPrompt,
			},
			{
				Role:    "user",
				Content: &stepInput,
			},
		},
	}

	ai_response, code, err := telexai.TranslatorCompletions(logger, extReq, req)
	if err != nil {
		logger.Error("Error Generating Translator Completions: %v\n", err)
		return "", code, fmt.Errorf("error generating translator completions: %v", err)
	}

	response, err := telexai.ExtractChatContent(ai_response)
	if err != nil {
		return "", http.StatusInternalServerError, err
	}

	return response, code, nil
}
