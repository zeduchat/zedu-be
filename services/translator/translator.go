package translator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/telexai"
	"github.com/hngprojects/telex_be/utility"
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
		"skills": strings.Join(req.Skills, ", "),
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
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: stepInput,
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

func GenerateWorkflowJSON(db *gorm.DB, logger *utility.Logger, extReq request.ExternalRequest, agentID string) (models.WorkflowJSON, int, error) {
	var (
		agents      models.OrganisationIntegrations
		task        models.Task
		skillsModel models.AgentSkill
		aw          models.AgentWorkflow
	)

	exists, err := agents.CheckAgentExists(db, agentID)
	if !exists {
		return models.WorkflowJSON{}, http.StatusNotFound, fmt.Errorf("agent with id %s not found", agentID)
	}
	if err != nil {
		return models.WorkflowJSON{}, http.StatusInternalServerError, err
	}

	tasks, err := (task).GetAgentTasks(db, agentID)
	if err != nil {
		return models.WorkflowJSON{}, http.StatusInternalServerError, err
	}

	if len(tasks) == 0 {
		logger.Error("no tasks found for agent id %s", agentID)
		return models.WorkflowJSON{}, http.StatusOK, nil
	}

	var taskList strings.Builder
	for _, t := range tasks {
		taskList.WriteString(fmt.Sprintf("%s\n", t.Text))
	}

	skillsModel.AgentId = agentID
	skills, err := skillsModel.GetAllAgentSkills(db)
	if err != nil {
		return models.WorkflowJSON{}, http.StatusInternalServerError, err
	}
	if len(skills) == 0 {
		logger.Error("no skills found for agent id %s", agentID)
		return models.WorkflowJSON{}, http.StatusOK, nil
	}

	skillsList := make([]string, len(skills))
	for i, skill := range skills {
		skillsList[i] = skill.Name
	}

	promptSteps := []string{"Task Cleanup", "Skill Matching", "Workflow Translation"}
	steps := make([]models.StepReq, len(promptSteps))
	for i, step := range promptSteps {
		var prompt models.Prompts
		err := prompt.GetLatestPromptVersionByName(db, step)
		if err != nil {
			return models.WorkflowJSON{}, http.StatusInternalServerError, err
		}

		steps[i] = models.StepReq{
			Name:    step,
			Version: prompt.Version,
		}
	}

	req := models.TranslationRequest{
		TaskList: taskList.String(),
		Skills:   skillsList,
		Steps:    steps,
	}

	stepProcess, err := runTranslationPipeline(db, logger, extReq, taskList.String(), req)
	if err != nil {
		return models.WorkflowJSON{}, http.StatusBadRequest, err
	}

	resp := models.TranslationResponse{
		Status:      "success",
		ProcessStep: stepProcess,
	}

	wkfJson, err := ConvertToJSONObject(resp.ProcessStep[len(resp.ProcessStep)-1].Output)
	if err != nil {
		return models.WorkflowJSON{}, http.StatusInternalServerError, err
	}

	bytes, err := json.Marshal(wkfJson)
	if err != nil {
		return models.WorkflowJSON{}, http.StatusBadRequest, err
	}

	var rawEntry models.JSONBMap
	if err := json.Unmarshal(bytes, &rawEntry); err != nil {
		return models.WorkflowJSON{}, http.StatusBadRequest, err
	}

	aw.AgentId = agentID
	aw.ID = utility.GenerateUUID()
	aw.WorkflowId = utility.GenerateUUID()
	aw.RawEntry = rawEntry
	aw.Name = wkfJson.Name
	aw.OrgId = agents.OrgID

	err, code := aw.CreateAgentWorkflow(db)
	if err != nil {
		return models.WorkflowJSON{}, code, err
	}

	return wkfJson, http.StatusOK, nil
}

func ConvertToJSONObject(workflowStr string) (models.WorkflowJSON, error) {
	cleaned := strings.TrimSpace(workflowStr)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var workflow models.WorkflowJSON
	if err := json.Unmarshal([]byte(cleaned), &workflow); err != nil {
		return models.WorkflowJSON{}, fmt.Errorf("failed to parse workflow JSON: %w", err)
	}

	return workflow, nil
}
