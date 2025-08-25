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

func UpdateAgentTasks(c *gin.Context, db *gorm.DB, logger *utility.Logger, extReq request.ExternalRequest, req models.UpdateAgentTasksRequest) (int, []models.AgentSkillResponse, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return http.StatusInternalServerError, []models.AgentSkillResponse{}, tx.Error
	}

	var existingTasks []models.Task
	err := postgresql.SelectAllFromDb(tx, "", &existingTasks, "agent_id = ?", req.AgentID)
	if err != nil {
		tx.Rollback()
		return http.StatusInternalServerError, []models.AgentSkillResponse{}, err
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
				return http.StatusNotFound, []models.AgentSkillResponse{}, errors.New("task not found: " + *taskReq.ID)
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
				return http.StatusInternalServerError, []models.AgentSkillResponse{}, err
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
				return http.StatusInternalServerError, []models.AgentSkillResponse{}, err
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return http.StatusInternalServerError, []models.AgentSkillResponse{}, err
	}

	var gas models.GeneralAgentSkill
	generalskills, err, statusCode := gas.FetchGeneralAgentSkills(db, c)
	if err != nil {
		return statusCode, []models.AgentSkillResponse{}, err
	}

	allRecommendedSkills, err := GetRecommendedSkills(db, extReq, logger, updatedTasks, generalskills, req.AgentID)
	if err != nil {
		logger.Error("Failed to get recommended agent workflow skills: ", err)
		return http.StatusOK, []models.AgentSkillResponse{}, nil
	}

	recommendedSkills := make([]models.GeneralAgentSkill, 0)
	for _, rs := range allRecommendedSkills {
		var (
			gas models.GeneralAgentSkill
			as  models.AgentSkill
		)

		exists, err := as.CheckAgentHasSkillByName(db, req.AgentID, rs.Name)
		if err != nil {
			logger.Error("Failed to check if agent has skill: ", err)
			continue
		}

		if !exists {
			gas.GetGeneralAgentSkillByID(db, rs.ID)
			recommendedSkills = append(recommendedSkills, gas)
		}
	}


	if len(recommendedSkills) > 0 {
		err = StoreAgentSkills(db, logger, recommendedSkills, req.AgentID)
		if err != nil {
			logger.Error("Failed to store agent workflow skills: ", err)
			return http.StatusOK, allRecommendedSkills, nil
		}
	}

	return http.StatusOK, allRecommendedSkills, nil
}

func GetRecommendedSkills(db *gorm.DB, extReq request.ExternalRequest, logger *utility.Logger, tasksList []models.Task, generalSkills []models.GeneralAgentSkill, agentID string) ([]models.AgentSkillResponse, error) {
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
				ID:           skillID,
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
			logger.Error("Invalid skill ID recommended: ", skillID)
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
        logger.Info(fmt.Sprintf("Processing skill: %s", skill.Name))

        newSkill := models.AgentSkill{
            ID:           skill.ID,
            Name:         skill.Name,
            AgentId:      agentID,
            IsActive:     skill.IsActive,
            Description:  skill.Description,
            Type:         skill.Type,
            IsConfigured: skill.IsConfigured,
            Config:       skill.Config,
            Avatar:       skill.Avatar,
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
	if len(tasks) == 0 {
		logger.Info("No tasks found for agent ID: ", agentID)
		return nil, http.StatusNotFound, fmt.Errorf("no tasks found for agent ID: %s", agentID)
	}

	return tasks, http.StatusOK, nil
}
