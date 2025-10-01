package agents

import (
	"errors"
	"net/http"

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
			"workflow_json": map[string]any{},
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

	wkfJSON, statusCode, err := translator.GenerateWorkflowJSON(db, logger, extReq, ids, tasks, generalSkills)
	if err != nil {
		logger.Error("failed to generate workflow json", err)
		return statusCode, gin.H{
			"workflow_json": wkfJSON,
		}, nil
	}

	return http.StatusOK, gin.H{
		"workflow_json": wkfJSON,
	}, nil
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
