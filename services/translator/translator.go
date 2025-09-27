package translator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/telexai"
	"github.com/hngprojects/telex_be/utility"
)

func runTranslationPipeline(db *gorm.DB, logger *utility.Logger, extReq request.ExternalRequest, tasklist string, req models.TranslationRequest, agentID, orgID, userID string) ([]models.ProcessStep, error) {
	stepProcess := []models.ProcessStep{}
	placeholders := map[string]string{
		// "skills": strings.Join(req.Skills, ", "),
	}

	var previousOutput string = tasklist

	for _, step := range req.Steps {
		var prompt models.Prompts

		if _, err := prompt.GetPromptByVersion(db, step); err != nil {
			logger.Error(fmt.Sprintf("runTranslationPipeline: error getting prompt by version for step %s: %v", step.Name, err))
			return stepProcess, err
		}

		pStep := models.ProcessStep{
			Step:         step.Name,
			Input:        previousOutput,
			Status:       "in_progress",
			LLMCall:      true,
			SystemPrompt: prompt.Template,
		}

		switch step.Name {
		case "Task Cleanup":
			if pStep.LLMCall {
				stepInput := assemblePrompt(pStep, placeholders)
				aiOutput, _, err := LLMCall(logger, extReq, pStep.SystemPrompt, stepInput)
				if err != nil {
					logger.Error(fmt.Sprintf("runTranslationPipeline: error in Task Cleanup LLMCall: %v", err))
					return stepProcess, err
				}
				pStep.Output = aiOutput
			} else {
				pStep.Output = pStep.Input
			}

		case "Skill Matching":
			output, err := handleSkillMatching(db, logger, extReq, agentID, orgID, userID, pStep)
			if err != nil {
				logger.Error(fmt.Sprintf("runTranslationPipeline: error in Skill Matching: %v", err))
				return stepProcess, err
			}
			pStep.Output = output
			pStep.LLMCall = true

		case "Workflow Translation":
			if pStep.LLMCall {
				stepInput := assemblePrompt(pStep, placeholders)
				aiOutput, _, err := LLMCall(logger, extReq, pStep.SystemPrompt, stepInput)
				if err != nil {
					logger.Error(fmt.Sprintf("runTranslationPipeline: error in Workflow Translation LLMCall: %v", err))
					return stepProcess, err
				}
				pStep.Output = aiOutput
			} else {
				pStep.Output = pStep.Input
			}

		default:
			if pStep.LLMCall {
				stepInput := assemblePrompt(pStep, placeholders)
				aiOutput, _, err := LLMCall(logger, extReq, pStep.SystemPrompt, stepInput)
				if err != nil {
					logger.Error(fmt.Sprintf("runTranslationPipeline: error in default LLMCall for step %s: %v", step.Name, err))
					return stepProcess, err
				}
				pStep.Output = aiOutput
			} else {
				pStep.Output = pStep.Input
			}
		}

		previousOutput = pStep.Output
		pStep.Status = "completed"
		stepProcess = append(stepProcess, pStep)
	}

	return stepProcess, nil
}

func handleSkillMatching(db *gorm.DB, logger *utility.Logger, extReq request.ExternalRequest, agentID, orgID, userID string, pStep models.ProcessStep) (string, error) {
	cleanedTasks := pStep.Input

	var gas models.GeneralAgentSkill
	generalSkills, err, statusCode := gas.FetchGeneralAgentSkills(db, &gin.Context{})
	if err != nil {
		logger.Error(fmt.Sprintf("handleSkillMatching: Failed to fetch general agent skills: %v (status: %d)", err, statusCode))
		return "", fmt.Errorf("failed to fetch general skills: %v (status: %d)", err, statusCode)
	}

	skills := make([]models.SkillInfo, len(generalSkills))
	for i, skill := range generalSkills {
		skills[i] = models.SkillInfo{
			ID:          skill.ID,
			Name:        skill.Name,
			Description: skill.Description,
			Type:        skill.Type,
		}
	}

	skillsJSON, err := json.Marshal(skills)
	if err != nil {
		logger.Error(fmt.Sprintf("handleSkillMatching: failed to marshal skills: %v", err))
		return "", fmt.Errorf("failed to marshal skills: %v", err)
	}

	input := fmt.Sprintf("Skills: %s\n\nCleaned tasks: %s", string(skillsJSON), cleanedTasks)

	aiOutput, _, err := LLMCall(logger, extReq, pStep.SystemPrompt, input)
	if err != nil {
		logger.Error(fmt.Sprintf("handleSkillMatching: failed to get skill matching from LLM: %v", err))
		return "", fmt.Errorf("failed to get skill matching from LLM: %v", err)
	}
	fmt.Println("=============================", aiOutput)
	var skillMatchingResult []map[string]any
	if err := json.Unmarshal([]byte(aiOutput), &skillMatchingResult); err != nil {
		logger.Error(fmt.Sprintf("handleSkillMatching: failed to unmarshal skill matching result: %v", err))
		start := strings.Index(aiOutput, "[")
		end := strings.LastIndex(aiOutput, "]")
		if start != -1 && end != -1 && end > start {
			jsonStr := aiOutput[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &skillMatchingResult); err != nil {
				logger.Error(fmt.Sprintf("handleSkillMatching: failed to parse skill matching result from substring: %v", err))
				return "", fmt.Errorf("failed to parse skill matching result: %v", err)
			}
		} else {
			logger.Error(fmt.Sprintf("handleSkillMatching: failed to parse skill matching result, could not find valid JSON array: %v", err))
			return "", fmt.Errorf("failed to parse skill matching result: %v", err)
		}
	}

	if err := storeRecommendedSkills(db, logger, skillMatchingResult, generalSkills, agentID, orgID, userID); err != nil {
		logger.Error(fmt.Sprintf("handleSkillMatching: Failed to store recommended skills: %v", err))
	}

	return aiOutput, nil
}

func storeRecommendedSkills(db *gorm.DB, logger *utility.Logger, skillMatchingResult []map[string]any, generalSkills []models.GeneralAgentSkill, agentID, orgID, userID string) error {
	uniqueSkillIDs := make(map[string]bool)

	for _, result := range skillMatchingResult {

		if result["unsupported"].(bool) {
			continue
		}

		for _, matchedSkills := range result["matched_skills"].([]any) {
			uniqueSkillIDs[matchedSkills.(map[string]any)["skill_id"].(string)] = true
		}
	}

	generalSkillMap := make(map[string]models.GeneralAgentSkill, 0)
	for _, generalSkill := range generalSkills {
		generalSkillMap[generalSkill.ID] = generalSkill
	}

	recommendedSkills := make([]models.AgentSkill, 0)
	for skillID := range uniqueSkillIDs {
		if generalSkill, exists := generalSkillMap[skillID]; exists {
			var as models.AgentSkill

			hasSkill, err := as.CheckAgentHasSkillByName(db, agentID, orgID, generalSkill.Name)
			if err != nil {
				logger.Error(fmt.Sprintf("storeRecommendedSkills: Error checking if agent has skill %s: %v", generalSkill.Name, err))
				continue
			}

			if !hasSkill {
				recommendedSkills = append(recommendedSkills, models.AgentSkill{
					ID:          utility.GenerateUUID(),
					SkillId:     skillID,
					AgentId:     agentID,
					OrgId:       orgID,
					UserId:      userID,
					Name:        generalSkill.Name,
					Description: generalSkill.Description,
					Type:        generalSkill.Type,
					Avatar:      generalSkill.Avatar,
					Config:      generalSkill.Config,
					IsActive:    true,
				})
			}
		} else {
			logger.Error(fmt.Sprintf("storeRecommendedSkills: Invalid skill ID recommended by LLM: %s", skillID))
		}
	}

	if len(recommendedSkills) > 0 {
		if err := StoreAgentSkills(db, logger, recommendedSkills, agentID, orgID, userID); err != nil {
			logger.Error(fmt.Sprintf("storeRecommendedSkills: error storing agent skills: %v", err))
			return err
		}
	}

	return nil
}

func StoreAgentSkills(db *gorm.DB, logger *utility.Logger, skillsToAdd []models.AgentSkill, agentID, orgID, userID string) error {

	if len(skillsToAdd) == 0 {
		logger.Info("StoreAgentSkills: No skills to process")
		return nil
	}

	var existingSkillNames []string
	err := db.Model(&models.AgentSkill{}).
		Where("agent_id = ?", agentID).
		Pluck("name", &existingSkillNames).Error
	if err != nil {
		logger.Error(fmt.Sprintf("StoreAgentSkills: Failed to fetch existing agent skills: %v", err))
		return fmt.Errorf("failed to fetch existing agent skills: %v", err)
	}

	existingNames := make(map[string]bool)
	for _, name := range existingSkillNames {
		existingNames[name] = true
	}

	var newSkills []models.AgentSkill
	skippedCount := 0

	for _, skill := range skillsToAdd {
		if existingNames[skill.Name] {
			skippedCount++
			continue
		}
		newSkills = append(newSkills, skill)
	}

	logger.Info(fmt.Sprintf("StoreAgentSkills: Found %d new skills to add, %d already exist", len(newSkills), skippedCount))

	if len(newSkills) == 0 {
		logger.Info("StoreAgentSkills: No new skills to add - all skills already exist")
		return nil
	}

	tx := db.Begin()
	if tx.Error != nil {
		logger.Error(fmt.Sprintf("StoreAgentSkills: Failed to begin transaction: %v", tx.Error))
		return tx.Error
	}

	txResult := tx.CreateInBatches(&newSkills, len(newSkills))
	if txResult.Error != nil {
		logger.Error(fmt.Sprintf("StoreAgentSkills: Failed to save agent skills: %v", txResult.Error))
		tx.Rollback()
		return fmt.Errorf("failed to save agent skills: %v", txResult.Error)
	}

	if err := tx.Commit().Error; err != nil {
		logger.Error(fmt.Sprintf("StoreAgentSkills: Failed to commit transaction: %v", err))
		return fmt.Errorf("failed to commit agent skills transaction: %v", err)
	}

	logger.Info(fmt.Sprintf("StoreAgentSkills: Successfully stored %d new agent skills", len(newSkills)))
	return nil
}

func assemblePrompt(step models.ProcessStep, placeholderValues map[string]string) string {
	stepPrompt := step.Input

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
		logger.Error(fmt.Sprintf("LLMCall: Error Generating Translator Completions: %v", err))
		return "", code, fmt.Errorf("error generating translator completions: %v", err)
	}

	response, err := telexai.ExtractChatContent(ai_response)
	if err != nil {
		logger.Error(fmt.Sprintf("LLMCall: Error extracting chat content: %v", err))
		return "", http.StatusInternalServerError, err
	}

	return response, code, nil
}

func GenerateWorkflowJSON(db *gorm.DB, logger *utility.Logger, extReq request.ExternalRequest, ids models.IDS, tasks models.Tasks, generalSkills models.GeneralAgentSkills) (models.WorkflowJSON, int, error) {
	var (
		agents models.OrganisationIntegrations
		aw     models.AgentWorkflow
	)

	exists, err := agents.CheckAgentExists(db, ids.AgentID, ids.OrganisationID)
	if !exists {
		logger.Error(fmt.Sprintf("GenerateWorkflowJSON: agent with id %s not found", ids.AgentID))
		return models.WorkflowJSON{}, http.StatusNotFound, fmt.Errorf("agent with id %s not found", ids.AgentID)
	}
	if err != nil {
		logger.Error(fmt.Sprintf("GenerateWorkflowJSON: error checking agent exists: %v", err))
		return models.WorkflowJSON{}, http.StatusInternalServerError, err
	}

	if len(tasks) == 0 {
		logger.Error(fmt.Sprintf("GenerateWorkflowJSON: no tasks found for agent id %s", ids.AgentID))
		return models.WorkflowJSON{}, http.StatusNotFound, nil
	}

	var taskList strings.Builder
	for _, t := range tasks {
		taskList.WriteString(fmt.Sprintf("%s\n", t.Text))
	}

	promptSteps := []string{"Task Cleanup", "Skill Matching", "Workflow Translation"}
	steps := make([]models.StepReq, len(promptSteps))
	for i, step := range promptSteps {
		var prompt models.Prompts
		err := prompt.GetLatestPromptVersionByName(db, step)
		if err != nil {
			logger.Error(fmt.Sprintf("GenerateWorkflowJSON: error getting latest prompt version for step %s: %v", step, err))
			return models.WorkflowJSON{}, http.StatusInternalServerError, err
		}

		steps[i] = models.StepReq{
			Name:    step,
			Version: prompt.Version,
		}
	}

	req := models.TranslationRequest{
		TaskList: taskList.String(),
		Steps:    steps,
	}

	stepProcess, err := runTranslationPipeline(db, logger, extReq, taskList.String(), req, ids.AgentID, ids.OrganisationID, ids.UserID)
	if err != nil {
		logger.Error(fmt.Sprintf("GenerateWorkflowJSON: error running translation pipeline: %v", err))
		return models.WorkflowJSON{}, http.StatusBadRequest, err
	}

	resp := models.TranslationResponse{
		ProcessStep: stepProcess,
	}

	wkfJson, err := ConvertToJSONObject(resp.ProcessStep[len(resp.ProcessStep)-1].Output)
	if err != nil {
		logger.Error(fmt.Sprintf("GenerateWorkflowJSON: error converting to workflow JSON: %v", err))
		return models.WorkflowJSON{}, http.StatusInternalServerError, err
	}

	bytes, err := json.Marshal(wkfJson)
	if err != nil {
		logger.Error(fmt.Sprintf("GenerateWorkflowJSON: error marshalling workflow JSON: %v", err))
		return models.WorkflowJSON{}, http.StatusBadRequest, err
	}

	var rawEntry models.JSONBMap
	if err := json.Unmarshal(bytes, &rawEntry); err != nil {
		logger.Error(fmt.Sprintf("GenerateWorkflowJSON: error unmarshalling workflow JSON: %v", err))
		return models.WorkflowJSON{}, http.StatusBadRequest, err
	}

	aw.AgentId = ids.AgentID
	aw.ID = utility.GenerateUUID()
	aw.WorkflowId = utility.GenerateUUID()
	aw.RawEntry = rawEntry
	aw.Name = wkfJson.Name
	aw.OrgId = ids.OrganisationID

	err, code := aw.CreateAgentWorkflow(db)
	if err != nil {
		logger.Error(fmt.Sprintf("GenerateWorkflowJSON: error creating agent workflow: %v", err))
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
