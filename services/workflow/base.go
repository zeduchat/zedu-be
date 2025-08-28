package workflow

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
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

func AddWorkflowToChannel(db *gorm.DB, req models.ChannelWorkflowRequest) (int, error) {

	cw := models.ChannelWorkflow{
		ID:         utility.GenerateUUID(),
		ChannelID:  req.ChannelID,
		WorkflowID: req.WorkflowID,
	}
	status, err := cw.Add(db)
	if err != nil {
		return status, fmt.Errorf("failed to add workflow to channel: %v", err)
	}
	return status, nil
}

func RemoveWorkflowFromChannel(db *gorm.DB, req models.ChannelWorkflowRequest) (int, error) {
	cw := models.ChannelWorkflow{
		ChannelID:  req.ChannelID,
		WorkflowID: req.WorkflowID,
	}

	status, err := cw.RemoveChannelWorkflow(db)
	if err != nil {
		return status, fmt.Errorf("failed to remove workflow from channel: %v", err)
	}
	return status, nil
}

func GetChannelWorkflows(db *gorm.DB, channelID, orgId *string) ([]models.WorkflowSummary, int, error) {
	cw := &models.ChannelWorkflow{ChannelID: *channelID}
	workflows, err := cw.GetWorkflowsWithChannelStatus(db, orgId)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to retrieve workflows: %v", err)
	}
	return workflows, http.StatusOK, nil
}

func UpdateWorkflowStatus(db *gorm.DB, req models.ChannelWorkflowRequest) (int, error) {
	wc := models.ChannelWorkflow{
		ChannelID:  req.ChannelID,
		WorkflowID: req.WorkflowID,
	}

	exists, _ := wc.CheckChannelWorkflowExists(db)

	if exists {
		return RemoveWorkflowFromChannel(db, req)
	}

	return AddWorkflowToChannel(db, req)
}

func ListGeneralMarketPlaceWorkflows(db *gorm.DB, c *gin.Context) (*[]models.GeneralWorkflow, postgresql.PaginationResponse, int, error) {
	gw := models.GeneralWorkflow{}
	wfs, pag, err := gw.GetMarketPlaceWorkflows(db, c)
	if err != nil {
		return nil, postgresql.PaginationResponse{}, http.StatusInternalServerError, err
	}
	return wfs, pag, http.StatusOK, nil
}

func GetGeneralMarketPlaceWorkflowId(req models.WorkFlowRequest, db *gorm.DB) (*[]models.GeneralWorkflow, int, error) {
	gw := models.GeneralWorkflow{ID: req.Id}
	wfs, err := gw.GetMarketPlaceWorkflowById(db)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return wfs, http.StatusOK, nil
}

// Agent workflow request
func CreateAgentWorkflowService(req models.AgentWorkFlowRequest, db *gorm.DB) (*models.AgentWorkflow, int, error) {
	var wf models.AgentWorkflow

	wf.WorkflowId = utility.GenerateUUID()
	wf.AgemtId = req.AgentId
	wf.RawEntry = req.RawEntry
	wf.OrgId = req.OrgId

	err, code := wf.CreateAgentWorkflow(db)

	return &wf, code, err
}

// Get Workflow by ID Service
func GetAgentWorkflowByIDService(req models.AgentWorkFlowRequest, db *gorm.DB) (*models.AgentWorkFlowResponse, int, error) {
	var wf models.AgentWorkflow
	wf.OrgId = req.OrgId
	wf.AgemtId = req.AgentId
	wf.WorkflowId = req.WorkflowId

	res, code, err := wf.GetWorkflowByID(db)
	return res, code, err
}

// List Workflows Service
func ListAgentWorkflowsService(req models.AgentWorkFlowRequest, db *gorm.DB) (*[]models.AgentWorkflowSummary, int, error) {
	var wf models.AgentWorkflow
	wf.OrgId = req.OrgId
	wf.AgemtId = req.AgentId

	res, code, err := wf.ListWorkflows(db)
	return res, code, err
}

// Delete Workflow Service
func DeleteAgentWorkflowService(req models.AgentWorkFlowRequest, db *gorm.DB) (error, int) {
	var wf models.AgentWorkflow
	wf.OrgId = req.OrgId
	wf.AgemtId = req.AgentId
	wf.WorkflowId = req.WorkflowId

	err, code := wf.DeleteWorkflow(db)
	return err, code
}

// Update Workflow Service
func UpdateAgentWorkflowService(req models.AgentWorkFloUpdatewRequest, db *gorm.DB) (error, int) {
	var wf models.AgentWorkflow
	wf.OrgId = req.OrgId
	wf.AgemtId = req.AgentId
	wf.WorkflowId = req.WorkflowId
	wf.RawEntry = req.RawEntry
	wf.IsActive = req.IsActive

	err, code := wf.UpdateAgentWorkflow(db)
	return err, code
}
