package agents

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
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

func ProcessAgentTasks(c *gin.Context, db *gorm.DB, logger *utility.Logger, extReq request.ExternalRequest, agentID string) (int, gin.H, error) {

	tasks, code, err := GetAgentTasks(c, db, logger, agentID)
	if err != nil {
		logger.Error("error fetching tasks", err)
		return code, gin.H{}, err
	}

	if len(tasks) == 0 {
		logger.Info("No tasks found for agent")
		return http.StatusOK, gin.H{}, nil
	}

	var gas models.GeneralAgentSkill
	generalskills, err, statusCode := gas.FetchGeneralAgentSkills(db, c)
	if err != nil {
		return statusCode, gin.H{}, err
	}

	allRecommendedSkills, err := GetRecommendedSkills(db, extReq, logger, tasks, generalskills, agentID)
	if err != nil {
		logger.Error("Failed to get recommended agent workflow skills: ", err)
		return http.StatusOK, gin.H{}, nil
	}

	recommendedSkills := make([]models.GeneralAgentSkill, 0)
	for _, rs := range allRecommendedSkills {
		var (
			gas models.GeneralAgentSkill
			as  models.AgentSkill
		)

		exists, err := as.CheckAgentHasSkillByName(db, agentID, rs.Name)
		if err != nil {
			logger.Error(err)
			continue
		}

		if !exists {
			gas.GetGeneralAgentSkillByID(db, rs.SkillId)
			recommendedSkills = append(recommendedSkills, gas)
		}
	}

	if len(recommendedSkills) > 0 {
		err = StoreAgentSkills(db, logger, recommendedSkills, agentID)
		if err != nil {
			logger.Error("Failed to store agent workflow skills: ", err)
			resp := gin.H{
				"workflow_json":          map[string]any{},
				"all_recommended_skills": allRecommendedSkills,
			}
			return http.StatusOK, resp, nil
		}
	}

	//generate the workflow json
	wkfJSON, statusCode, err := translator.GenerateWorkflowJSON(db, logger, extReq, agentID)
	if err != nil {
		logger.Error("failed to generate workflow json: ", err)
		resp := gin.H{
			"all_recommended_skills": allRecommendedSkills,
			"workflow_json":          wkfJSON,
		}
		return statusCode, resp, nil
	}

	response := gin.H{
		"all_recommended_skills": allRecommendedSkills,
		"workflow_json":          wkfJSON,
	}

	return http.StatusOK, response, nil
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

		Return ONLY the JSON array, nothing else.`

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

func StoreAgentSkills(db *gorm.DB, logger *utility.Logger, recommendedskills []models.GeneralAgentSkill, agentID string) error {
	logger.Info(fmt.Sprintf("Attempting to store %d recommended skills", len(recommendedskills)))

	var as []models.AgentSkill
	if len(recommendedskills) == 0 {
		logger.Info("No recommended skills to store")
		return nil
	}

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
		}

		logger.Info(newSkill)
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
