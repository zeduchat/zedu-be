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
		ID:             utility.GenerateUUID(),
		AgentID:        req.AgentID,
		Text:           req.Text,
		OrganisationID: req.OrganisationID,
		Position:       req.Position,
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

	var agent models.OrganisationIntegrations
	if !postgresql.CheckExists(db, &agent, "integration_id = ? AND org_id = ?", ids.AgentID, ids.OrganisationID) {
		logger.Error("agent not found")
		return http.StatusNotFound, emptyResp(), errors.New("agent not found")
	}

	tasks, code, err := GetAgentTasks(c, db, logger, ids.AgentID, ids.OrganisationID)
	if err != nil {
		logger.Error("error fetching tasks", err)
		return code, emptyResp(), err
	}
	if len(tasks) == 0 {
		logger.Error("no tasks found for agent")
		return http.StatusBadRequest, emptyResp(), errors.New("no tasks found for agent")
	}

	var gas models.GeneralAgentSkill
	generalSkills, err, statusCode := gas.FetchGeneralAgentSkills(db, c)
	if err != nil {
		logger.Error("error fetching general agent skills", err)
		return statusCode, emptyResp(), err
	}

	allRecommendedSkills, err := GetRecommendedSkills(db, extReq, logger, tasks, generalSkills, ids.AgentID)
	if err != nil {
		logger.Error("failed to get recommended agent workflow skills", err)
		return http.StatusInternalServerError, emptyResp(), err
	}

	skillsToStore := make([]models.AgentSkill, 0)
	for _, rs := range allRecommendedSkills {
		var g models.GeneralAgentSkill
		if err := g.GetGeneralAgentSkillByID(db, rs.SkillId); err != nil {
			logger.Error("error fetching general agent skill", err)
			continue
		}

		skillsToStore = append(skillsToStore, models.AgentSkill{
			ID:           utility.GenerateUUID(),
			Name:         g.Name,
			AgentId:      ids.AgentID,
			SkillId:      g.ID,
			Description:  g.Description,
			Type:         g.Type,
			IsActive:     g.IsActive,
			IsConfigured: g.IsConfigured,
			Avatar:       g.Avatar,
			Config:       g.Config,
			OrgId:        ids.OrganisationID,
			UserId:       ids.UserID,
		})
	}

	if len(skillsToStore) > 0 {
		if err := StoreAgentSkills(db, logger, skillsToStore, ids.AgentID, ids.OrganisationID, ids.UserID); err != nil {
			logger.Error("failed to store agent workflow skills", err)
			return http.StatusInternalServerError, emptyResp(), err
		}
	}

	wkfJSON, statusCode, err := translator.GenerateWorkflowJSON(db, logger, extReq, ids.AgentID, ids.OrganisationID)
	if err != nil {
		logger.Error("failed to generate workflow json", err)
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

func StoreAgentSkills(db *gorm.DB, logger *utility.Logger, skillsToAdd []models.AgentSkill, agentID, orgID, userID string) error {

	if len(skillsToAdd) == 0 {
		logger.Info("No skills to process")
		return nil
	}

	var existingSkillNames []string
	err := db.Model(&models.AgentSkill{}).
		Where("agent_id = ?", agentID).
		Pluck("name", &existingSkillNames).Error
	if err != nil {
		logger.Error("Failed to fetch existing agent skills:", err)
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

	logger.Info(fmt.Sprintf("Found %d new skills to add, %d already exist", len(newSkills), skippedCount))

	if len(newSkills) == 0 {
		logger.Info("No new skills to add - all skills already exist")
		return nil
	}

	tx := db.Begin()
	if tx.Error != nil {
		logger.Error("Failed to begin transaction:", tx.Error)
		return tx.Error
	}

	txResult := tx.CreateInBatches(&newSkills, len(newSkills))
	if txResult.Error != nil {
		logger.Error("Failed to save agent skills:", txResult.Error)
		tx.Rollback()
		return fmt.Errorf("failed to save agent skills: %v", txResult.Error)
	}

	if err := tx.Commit().Error; err != nil {
		logger.Error("Failed to commit transaction:", err)
		return fmt.Errorf("failed to commit agent skills transaction: %v", err)
	}

	logger.Info(fmt.Sprintf("Successfully stored %d new agent skills", len(newSkills)))
	return nil
}

func GetAgentTasks(c *gin.Context, db *gorm.DB, logger *utility.Logger, agentID, orgID string) ([]models.Task, int, error) {
	var task models.Task

	tasks, err := task.GetAgentTasks(db, agentID, orgID)
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
