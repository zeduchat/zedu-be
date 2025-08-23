package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/translator"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func UpdateAgentTasks(c *gin.Context, db *gorm.DB, logger *utility.Logger, extReq request.ExternalRequest, req models.UpdateAgentTasksRequest) (int, map[string]any, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return http.StatusInternalServerError, nil, tx.Error
	}

	var existingTasks []models.Task
	err := postgresql.SelectAllFromDb(tx, "", &existingTasks, "agent_id = ?", req.AgentID)
	if err != nil {
		tx.Rollback()
		return http.StatusInternalServerError, nil, err
	}

	existingMap := make(map[string]models.Task)
	for _, t := range existingTasks {
		existingMap[t.ID] = t
	}

	incomingIDs := make(map[string]bool)
	var updatedTasks []models.Task

	for _, taskReq := range req.Tasks {
		if taskReq.ID != nil {
			existingTask, ok := existingMap[*taskReq.ID]
			if !ok {
				tx.Rollback()
				return http.StatusNotFound, nil, errors.New("task not found: " + *taskReq.ID)
			}

			changed := false
			if existingTask.Text != taskReq.Text {
				existingTask.Text = taskReq.Text
				changed = true
			}
			if existingTask.Position != taskReq.Position {
				existingTask.Position = taskReq.Position
				changed = true
			}

			if changed {
				if err := tx.Save(&existingTask).Error; err != nil {
					tx.Rollback()
					return http.StatusInternalServerError, nil, err
				}
			}

			incomingIDs[existingTask.ID] = true
			updatedTasks = append(updatedTasks, existingTask)

		} else {
			newTask := models.Task{
				ID:       utility.GenerateUUID(),
				AgentID:  req.AgentID,
				Text:     taskReq.Text,
				Position: taskReq.Position,
			}
			if err := tx.Create(&newTask).Error; err != nil {
				tx.Rollback()
				return http.StatusInternalServerError, nil, err
			}
			incomingIDs[newTask.ID] = true
			updatedTasks = append(updatedTasks, newTask)
		}
	}

	for id := range existingMap {
		if !incomingIDs[id] {
			err := postgresql.HardDeleteSpecificRecord(tx, &models.Task{}, "id = ?", id)
			if err != nil {
				tx.Rollback()
				return http.StatusInternalServerError, nil, err
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return http.StatusInternalServerError, nil, err
	}

	var gas models.GeneralAgentSkill
	generalskills, err, statusCode := gas.FetchGeneralAgentSkills(db, c)
	if err != nil {
		return statusCode, nil, err
	}

	allRecommendedSkills, err := GetRecommendedSkills(extReq, logger, updatedTasks, generalskills, req.AgentID)
	if err != nil {
		logger.Error("Failed to get recommended agent workflow skills: ", err)
		return http.StatusOK, nil, nil
	}

	recommendedSkills := make([]models.GeneralAgentSkill, 0)
	currentSkills := make([]models.AgentSkillResponse, 0)
	for _, rs := range allRecommendedSkills {
		var (
			gas models.GeneralAgentSkill
			as  models.AgentSkill
		)

		exists, err := as.CheckAgentHasSkill(db, req.AgentID, rs.SkillID)
		if err != nil {
			logger.Error("Failed to check if agent has skill: ", err)
			continue
		}

		if exists {
			as.ID = rs.SkillID
			as.AgentId = req.AgentID
			resp, err := as.GetAgentSkillByID(db)
			if err != nil {
				logger.Error("Failed to get agent skill by ID: ", err)
				continue
			}
			currentSkills = append(currentSkills, resp)
		} else {
			gas.GetGeneralAgentSkillByID(db, rs.SkillID)
			recommendedSkills = append(recommendedSkills, gas)
		}
	}

	resp := map[string]any{
		"recommended_skills": recommendedSkills,
		"current_skills":     currentSkills,
	}

	// err = StoreWorkflowSkills(db, logger, allRecommendedSkills)
	// if err != nil {
	// 	logger.Error("Failed to store agent workflow skills: ", err)
	// 	return http.StatusOK, updatedTasks, nil
	// }

	return http.StatusOK, resp, nil
}

func GetRecommendedSkills(extReq request.ExternalRequest, logger *utility.Logger, tasksList []models.Task, generalSkills []models.GeneralAgentSkill, agentID string) ([]models.SkillResp, error) {
	type skillInfo struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	skills := make([]skillInfo, len(generalSkills))
	for i, skill := range generalSkills {
		skills[i] = skillInfo{
			ID:   skill.ID,
			Name: skill.Name,
		}
	}

	var taskDescriptions strings.Builder
	for _, task := range tasksList {
		taskDescriptions.WriteString(fmt.Sprintf("Task ID: %s, Task: %s\n", task.ID, task.Text))
	}

	systemPrompt := `You are a skill recommendation assistant. Given a list of tasks and available skills, identify which skills are relevant to the overall workflow. 
	Return ONLY a valid JSON array containing UNIQUE skill IDs that are relevant to the workflow:
	["skill-id-1", "skill-id-2", "skill-id-3"]
	Only return the JSON array, nothing else.`

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

	var skillResp []models.SkillResp
	skillMap := make(map[string]bool)
	for _, skill := range generalSkills {
		skillMap[skill.ID] = true
	}

	uniqueSkills := make(map[string]bool)

	for _, skillID := range recommendedSkillIDs {
		if skillMap[skillID] && !uniqueSkills[skillID] {
			skillResp = append(skillResp, models.SkillResp{
				AgentID: agentID,
				SkillID: skillID,
			})
			uniqueSkills[skillID] = true
		} else if !skillMap[skillID] {
			logger.Error("Invalid skill ID recommended: ", skillID)
		}
	}

	return skillResp, nil
}

// func StoreWorkflowSkills(db *gorm.DB, logger *utility.Logger, agentWorkflowSkills []models.AgentWorkflowSkills) error {
// 	if len(agentWorkflowSkills) == 0 {
// 		return nil
// 	}

// 	tx := db.Begin()
// 	if tx.Error != nil {
// 		return tx.Error
// 	}

// 	agentID := agentWorkflowSkills[0].AgentID

// 	err := tx.Where("agent_id = ?", agentID).Delete(&models.AgentWorkflowSkills{}).Error
// 	if err != nil {
// 		tx.Rollback()
// 		return fmt.Errorf("failed to clear existing agent workflow skills: %v", err)
// 	}

// 	err = tx.CreateInBatches(&agentWorkflowSkills, 100).Error
// 	if err != nil {
// 		tx.Rollback()
// 		return fmt.Errorf("failed to save agent workflow skills: %v", err)
// 	}

// 	if err := tx.Commit().Error; err != nil {
// 		return fmt.Errorf("failed to commit agent workflow skills transaction: %v", err)
// 	}

// 	logger.Info("Successfully stored %d agent workflow skills")
// 	return nil
// }

func GetAgentTasks(c *gin.Context, db *gorm.DB, logger *utility.Logger, agentID string) ([]models.Task, int, error) {
	var task models.Task

	tasks, err := task.GetAgentTasks(db, agentID)
	if err != nil {
		logger.Error("error fetching tasks", err)
		return nil, http.StatusInternalServerError, err
	}
	if len(tasks) == 0 {
		logger.Info("No tasks found for agent ID: ", agentID)
		return nil, http.StatusNotFound, fmt.Errorf("no tasks found for agent ID: %s", agentID)
	}

	return tasks, http.StatusOK, nil
}
