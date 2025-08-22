package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/translator"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func UpdateWorkflowTasks(c *gin.Context, db *gorm.DB, logger *utility.Logger, extReq request.ExternalRequest, req models.UpdateWorkflowTasksRequest) (int, []models.Task, []models.WorkflowSkills, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return http.StatusInternalServerError, nil, nil, tx.Error
	}

	var existingTasks []models.Task
	err := postgresql.SelectAllFromDb(tx, "", &existingTasks, "workflow_id = ?", req.WorkflowID)
	if err != nil {
		tx.Rollback()
		return http.StatusInternalServerError, nil, nil, err
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
				return http.StatusNotFound, nil, nil, errors.New("task not found: " + *taskReq.ID)
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
					return http.StatusInternalServerError, nil, nil, err
				}
			}

			incomingIDs[existingTask.ID] = true
			updatedTasks = append(updatedTasks, existingTask)

		} else {
			newTask := models.Task{
				ID:         utility.GenerateUUID(),
				WorkflowID: req.WorkflowID,
				Text:       taskReq.Text,
				Position:   taskReq.Position,
			}
			if err := tx.Create(&newTask).Error; err != nil {
				tx.Rollback()
				return http.StatusInternalServerError, nil, nil, err
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
				return http.StatusInternalServerError, nil, nil, err
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return http.StatusInternalServerError, nil, nil, err
	}

	var gas models.GeneralAgentSkill
	generalskills, err, statusCode := gas.FetchGeneralAgentSkills(db, c)
	if err != nil {
		return statusCode, updatedTasks, nil, err
	}

	recommendedSkills, err := GetRecommendedSkills(extReq, logger, updatedTasks, generalskills, req.WorkflowID)
	if err != nil {
		logger.Error("Failed to get recommended skills: ", err)
		return http.StatusOK, updatedTasks, nil, nil
	}

	err = StoreWorkflowSkills(db, logger, recommendedSkills)
	if err != nil {
		logger.Error("Failed to store workflow skills: ", err)
		return http.StatusOK, updatedTasks, recommendedSkills, nil
	}

	return http.StatusOK, updatedTasks, recommendedSkills, nil
}

func GetRecommendedSkills(extReq request.ExternalRequest, logger *utility.Logger, tasksList []models.Task, generalSkills []models.GeneralAgentSkill, workflowID string) ([]models.WorkflowSkills, error) {
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

	var workflowSkills []models.WorkflowSkills
	skillMap := make(map[string]bool)
	for _, skill := range generalSkills {
		skillMap[skill.ID] = true
	}

	uniqueSkills := make(map[string]bool)
	
	for _, skillID := range recommendedSkillIDs {
		if skillMap[skillID] && !uniqueSkills[skillID] {
			workflowSkills = append(workflowSkills, models.WorkflowSkills{
				ID:         utility.GenerateUUID(),
				WorkflowID: workflowID,
				SkillID:    skillID,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			})
			uniqueSkills[skillID] = true
		} else if !skillMap[skillID] {
			logger.Error("Invalid skill ID recommended: ", skillID)
		}
	}

	return workflowSkills, nil
}

func StoreWorkflowSkills(db *gorm.DB, logger *utility.Logger, workflowSkills []models.WorkflowSkills) error {
	if len(workflowSkills) == 0 {
		return nil
	}

	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	workflowID := workflowSkills[0].WorkflowID

	err := tx.Where("workflow_id = ?", workflowID).Delete(&models.WorkflowSkills{}).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to clear existing workflow skills: %v", err)
	}

	err = tx.CreateInBatches(&workflowSkills, 100).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to save workflow skills: %v", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit workflow skills transaction: %v", err)
	}

	logger.Info("Successfully stored %d workflow skills for workflow %s", len(workflowSkills), workflowID)
	return nil
}


func GetWorkflowTasks(c *gin.Context, db *gorm.DB, logger *utility.Logger, workflowID string) ([]models.Task, int, error) {
	var task models.Task
	
	tasks, err := task.GetWorkflowTasks(db, workflowID)
	if err != nil {
		logger.Error("error fetching tasks", err)
		return nil, http.StatusInternalServerError, err
	}
	if len(tasks) == 0 {
		logger.Info("No tasks found for workflow ID: ", workflowID)
		return nil, http.StatusNotFound, fmt.Errorf("no tasks found for workflow ID: %s", workflowID)
	}

	return tasks, http.StatusOK, nil
}

func GetWorkflowSkills(c *gin.Context, db *gorm.DB, logger *utility.Logger, workflowID string) ([]models.WorkflowSkills, int, error) {
	var workflowSkills models.WorkflowSkills
	
	skills, err := workflowSkills.GetWorkflowSkills(db, workflowID)
	if err != nil {
		logger.Error("error fetching workflow skills", err)
		return nil, http.StatusInternalServerError, err
	}
	if len(skills) == 0 {
		logger.Info("No skills found for workflow ID: ", workflowID)
		return nil, http.StatusNotFound, fmt.Errorf("no skills found for workflow ID: %s", workflowID)
	}

	return skills, http.StatusOK, nil
}