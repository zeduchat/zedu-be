package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/translator"
	"github.com/hngprojects/telex_be/utility"
)

func CreateAgentTasks(db *gorm.DB, logger *utility.Logger, req models.CreateAgentTasksRequest) (int, models.Task, error) {

	task := models.Task{
		ID:       utility.GenerateUUID(),
		AgentID:  req.AgentID,
		Text:     req.Text,
		Position: req.Position,
	}

	code, err := task.CreateTasks(db)
	if err != nil {
		return code, models.Task{}, err
	}

	return http.StatusCreated, task, nil
}

func UpdateAgentTasks(db *gorm.DB, logger *utility.Logger, req models.UpdateAgentTasksRequest, ids models.IDS) (int, error) {

	var task models.Task

	code, err := task.UpdateAgentTasks(db, req, ids)
	if err != nil {
		return code, err
	}

	return http.StatusOK, nil
}

func ProcessAgentTasks(c *gin.Context, db *gorm.DB, logger *utility.Logger, extReq request.ExternalRequest, ids models.IDS) (int, gin.H, error) {

	emptyResp := func() gin.H {
		return gin.H{
			"workflow_json":          map[string]any{},
			"all_recommended_skills": []models.AgentSkillResponse{},
		}
	}

	fail := func(code int, msg string, err error) (int, gin.H, error) {
		if err != nil {
			logger.Error(msg, err)
		} else {
			logger.Error(msg)
		}
		return code, emptyResp(), err
	}

	var agent models.OrganisationIntegrations
	if !postgresql.CheckExists(db, &agent, "integration_id = ?", ids.AgentID) {
		return fail(http.StatusNotFound, "agent not found", nil)
	}

	tasks, code, err := GetAgentTasks(c, db, logger, ids.AgentID)
	if err != nil {
		return fail(code, "error fetching tasks", err)
	}
	if len(tasks) == 0 {
		return fail(http.StatusBadRequest, "No tasks found for agent", errors.New("no tasks found for agent"))
	}

	var gas models.GeneralAgentSkill
	generalSkills, err, statusCode := gas.FetchGeneralAgentSkills(db, c)
	if err != nil {
		return fail(statusCode, "error fetching general agent skills", err)
	}

	allRecommendedSkills, err := GetRecommendedSkills(db, extReq, logger, tasks, generalSkills, ids.AgentID)
	if err != nil {
		return fail(http.StatusOK, "Failed to get recommended agent workflow skills", err)
	}

	if len(allRecommendedSkills) == 0 {
		return fail(http.StatusBadRequest, "No Skills Recommended from global skills", errors.New("no skills recommended for agent from global skills"))
	}

	recommendedSkills := make([]models.GeneralAgentSkill, 0)
	for _, rs := range allRecommendedSkills {
		var as models.AgentSkill
		exists, err := as.CheckAgentHasSkillByName(db, ids.AgentID, rs.Name)
		if err != nil {
			logger.Error(err.Error())
			continue
		}
		if !exists {
			var g models.GeneralAgentSkill
			g.GetGeneralAgentSkillByID(db, rs.SkillId)
			recommendedSkills = append(recommendedSkills, g)
		}
	}

	if len(recommendedSkills) > 0 {
		if err := StoreAgentSkills(db, logger, recommendedSkills, ids.AgentID, ids.OrganisationID, ids.UserID); err != nil {
			return fail(http.StatusOK, "Failed to store agent workflow skills", err)
		}
	}

	wkfJSON, statusCode, err := translator.GenerateWorkflowJSON(db, logger, extReq, ids.AgentID, ids.OrganisationID)
	if err != nil {
		logger.Error("failed to generate workflow json: ", err)
		return statusCode, gin.H{
			"all_recommended_skills": allRecommendedSkills,
			"workflow_json":          wkfJSON,
		}, nil
	}

	return http.StatusOK, gin.H{
		"all_recommended_skills": allRecommendedSkills,
		"workflow_json":          wkfJSON,
	}, nil
}

func GetRecommendedSkills(db *gorm.DB, extReq request.ExternalRequest, logger *utility.Logger, tasksList []models.Task, generalSkills []models.GeneralAgentSkill, agentID string) ([]models.AgentSkillResponse, error) {
	type skillInfo struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        string `json:"type"`
	}

	skills := make([]skillInfo, len(generalSkills))
	for i, skill := range generalSkills {
		skills[i] = skillInfo{
			ID:          skill.ID,
			Name:        skill.Name,
			Description: skill.Description,
			Type:        skill.Type,
		}
	}

	var taskDescriptions strings.Builder
	for _, task := range tasksList {
		taskDescriptions.WriteString(fmt.Sprintf("Task ID: %s, Task: %s\n", task.ID, task.Text))
	}

	systemPrompt := `You are a skill recommendation assistant. 
		Given a list of tasks and available skills, identify which skills are DIRECTLY relevant to the overall workflow. 

		Rules:
		- Only return skills that CLEARLY and EXPLICITLY match the tasks. 
		- Do NOT infer or assume relevance if no clear match exists. 
		- If none of the available skills are relevant, return an EMPTY JSON array: []
		- The output must be a valid JSON array containing UNIQUE skill IDs only, for example:
		["skill-id-1", "skill-id-2"]

		Return ONLY the JSON array, nothing else. BE VERY ACCURATE`

	input := fmt.Sprintf("Skills: %+v\n\nTasks:\n%s", skills, taskDescriptions.String())

	aiResp, _, err := translator.LLMCall(logger, extReq, systemPrompt, input)
	if err != nil {
		return nil, err
	}

	var recommendedSkillIDs []string
	err = json.Unmarshal([]byte(aiResp), &recommendedSkillIDs)
	if err != nil {
		start := strings.Index(aiResp, "[")
		end := strings.LastIndex(aiResp, "]")
		if start != -1 && end != -1 && end > start {
			jsonStr := aiResp[start : end+1]
			err = json.Unmarshal([]byte(jsonStr), &recommendedSkillIDs)
			if err != nil {
				return nil, fmt.Errorf("failed to parse LLM response: %v", err)
			}
		} else {
			return nil, fmt.Errorf("failed to parse LLM response: %v", err)
		}
	}

	var skillResp []models.AgentSkillResponse
	skillMap := make(map[string]bool)
	for _, skill := range generalSkills {
		skillMap[skill.ID] = true
	}

	uniqueSkills := make(map[string]bool)

	for _, skillID := range recommendedSkillIDs {
		if skillMap[skillID] && !uniqueSkills[skillID] {
			var gas models.GeneralAgentSkill
			err := gas.GetGeneralAgentSkillByID(db, skillID)
			if err != nil {
				logger.Error("Failed to fetch general skill by ID: ", err)
				continue
			}
			skillResp = append(skillResp, models.AgentSkillResponse{
				SkillId:      skillID,
				Name:         gas.Name,
				Description:  gas.Description,
				Type:         gas.Type,
				IsActive:     gas.IsActive,
				IsConfigured: gas.IsConfigured,
				Avatar:       gas.Avatar,
				Config:       gas.Config,
			})
			uniqueSkills[skillID] = true
		} else if !skillMap[skillID] {
			logger.Error("Invalid skill ID recommended by llm: ", skillID)
		}
	}

	return skillResp, nil
}

func StoreAgentSkills(db *gorm.DB, logger *utility.Logger, recommendedskills []models.GeneralAgentSkill, agentID, orgID, userID string) error {

	logger.Info(fmt.Sprintf("Attempting to store %d recommended skills", len(recommendedskills)))
	var as []models.AgentSkill

	tx := db.Begin()
	if tx.Error != nil {
		logger.Error("Failed to begin transaction:", tx.Error)
		return tx.Error
	}

	for _, skill := range recommendedskills {
		logger.Info(fmt.Sprintf("Processing skill: %s, SkillID: %s ", skill.Name, skill.ID))

		newSkill := models.AgentSkill{
			ID:           utility.GenerateUUID(),
			SkillId:      skill.ID,
			AgentId:      agentID,
			Name:         skill.Name,
			IsActive:     skill.IsActive,
			Description:  skill.Description,
			Type:         skill.Type,
			IsConfigured: skill.IsConfigured,
			Config:       skill.Config,
			Avatar:       skill.Avatar,
			OrgId:        orgID,
			UserId:       userID,
		}

		as = append(as, newSkill)
	}

	if len(as) == 0 {
		logger.Error("No skills were processed for storage")
		tx.Rollback()
		return nil
	}

	err := tx.CreateInBatches(&as, 100).Error
	if err != nil {
		logger.Error("Failed to save agent skills:", err)
		tx.Rollback()
		return fmt.Errorf("failed to save agent skills: %v", err)
	}

	if err := tx.Commit().Error; err != nil {
		logger.Error("Failed to commit transaction:", err)
		return fmt.Errorf("failed to commit agent skills transaction: %v", err)
	}

	logger.Info(fmt.Sprintf("Successfully stored %d agent skills", len(as)))
	return nil
}

func GetAgentTasks(c *gin.Context, db *gorm.DB, logger *utility.Logger, agentID string) ([]models.Task, int, error) {
	var task models.Task

	tasks, err := task.GetAgentTasks(db, agentID)
	if err != nil {
		logger.Error("error fetching tasks", err)
		return nil, http.StatusInternalServerError, err
	}

	return tasks, http.StatusOK, nil
}

func DeleteAgentTasks(c *gin.Context, db *gorm.DB, logger *utility.Logger, ids models.IDS) (int, error) {
	var task models.Task

	code, err := task.DeleteAgentTasks(db, ids)
	if err != nil {
		return code, err
	}

	return http.StatusOK, nil
}
