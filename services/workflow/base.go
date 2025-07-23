package workflow

import (
	"errors"
	"net/http"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func CreateWorkflowService(req models.WorkFlowRequest, db *gorm.DB) (*models.Workflow, int, error) {
	var wf models.Workflow

	wf.ID = utility.GenerateUUID()
	wf.Name = req.Name
	wf.Description = req.Description
	wf.Tags = req.Tags
	wf.Meta = req.Meta
	wf.Agents = req.Agents
	wf.FlowConnections = req.FlowConnections
	wf.Settings = req.Settings
	wf.UserId = req.UserId
	wf.OrgId = req.OrgId

	if err := wf.CreateWorkflow(db); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return &wf, http.StatusCreated, nil
}

func GetWorkflowByIDService(req models.WorkFlowRequest, db *gorm.DB) (*models.WorkFlowResponse, int, error) {
	wf, err := models.GetWorkflowByID(db, req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusNotFound, err
		}
		return nil, http.StatusInternalServerError, err
	}
	return &wf, http.StatusOK, nil
}

func ListWorkflowsService(req models.WorkFlowRequest, db *gorm.DB) ([]models.WorkflowSummary, int, error) {
	wfs, err := models.ListWorkflows(db, req)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return wfs, http.StatusOK, nil
}

func DeleteWorkflowService(req models.WorkFlowRequest, db *gorm.DB) (int, error) {
	err := models.DeleteWorkflow(db, req)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func UpdateWorkflowService(req models.WorkFlowRequest, db *gorm.DB) (int, error) {
	wf := models.Workflow{
		ID:              req.Id,
		Name:            req.Name,
		Description:     req.Description,
		Tags:            req.Tags,
		Meta:            req.Meta,
		Agents:          req.Agents,
		FlowConnections: req.FlowConnections,
		Settings:        req.Settings,
		UserId:          req.UserId,
		OrgId:           req.OrgId,
	}

	err := wf.UpdateWorkflow(db)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
