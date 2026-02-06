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

	rawEntry := req.RawEntry

	shortDesc, ok := rawEntry["short_description"].(string)
	if !ok && req.ShortDescription == "" {
		return &models.AgentWorkflow{}, http.StatusUnprocessableEntity, errors.New("short_description is missing")
	}

	longDesc, ok := rawEntry["long_description"].(string)
	if !ok && req.LongDescription == "" {
		return &models.AgentWorkflow{}, http.StatusUnprocessableEntity, errors.New("long_description is missing")
	}

	category, ok := rawEntry["category"].(string)
	if !ok && req.Category == "" {
		return &models.AgentWorkflow{}, http.StatusUnprocessableEntity, errors.New("category is missing")
	}

	desc, ok := rawEntry["description"].(string)
	if !ok && req.Description == "" {
		return &models.AgentWorkflow{}, http.StatusUnprocessableEntity, errors.New("description is missing")
	}

	wf.WorkflowId = utility.GenerateUUID()
	wf.AgentId = req.AgentId
	wf.RawEntry = req.RawEntry
	wf.OrgId = req.OrgId
	wf.UserID = req.UserID
	wf.Name = req.Name

	wf.Description = req.Description
	if req.Description == "" {
		wf.Description = desc
	}

	wf.ShortDescription = req.ShortDescription
	if req.ShortDescription == "" {
		wf.ShortDescription = shortDesc
	}

	wf.LongDescription = req.LongDescription
	if req.LongDescription == "" {
		wf.LongDescription = longDesc
	}

	wf.IsActive = true
	wf.Category = req.Category
	if req.Category == "" {
		wf.Category = category
	}

	err, code := wf.CreateAgentWorkflow(db)
	return &wf, code, err
}

// Get Workflow by ID Service
func GetAgentWorkflowByIDService(req models.AgentWorkFlowRequest, db *gorm.DB) (*models.AgentWorkFlowResponse, int, error) {
	var wf models.AgentWorkflow
	wf.OrgId = req.OrgId
	wf.AgentId = req.AgentId
	wf.WorkflowId = req.WorkflowId

	res, code, err := wf.GetWorkflowByID(db)
	return res, code, err
}

// List Workflows Service
func ListAgentWorkflowsService(req models.AgentWorkFlowRequest, db *gorm.DB, c *gin.Context) ([]models.AgentWorkflowSummary, postgresql.PaginationResponse, int, error) {
	var wf models.AgentWorkflow
	wf.OrgId = req.OrgId
	wf.AgentId = req.AgentId
	wf.IsPublic = req.IsPublic

	res, pag, code, err := wf.ListWorkflows(db, c)
	return res, pag, code, err
}

// Delete Workflow Service
func DeleteAgentWorkflowService(req models.AgentWorkFlowRequest, db *gorm.DB) (error, int) {
	var wf models.AgentWorkflow
	wf.OrgId = req.OrgId
	wf.AgentId = req.AgentId
	wf.WorkflowId = req.WorkflowId

	err, code := wf.DeleteWorkflow(db)
	return err, code
}

// Update Workflow Service
func UpdateAgentWorkflowService(req models.AgentWorkFloUpdateRequest, db *gorm.DB) (error, int) {
	var wf models.AgentWorkflow
	wf.OrgId = req.OrgId
	wf.AgentId = req.AgentId
	wf.WorkflowId = req.WorkflowId
	wf.RawEntry = req.RawEntry
	wf.IsActive = req.IsActive
	wf.Name = req.Name

	err, code := wf.UpdateAgentWorkflow(db)
	return err, code
}

func UpdateWorkflowNodeService(req models.AgentWorkFloNodeUpdateRequest, db *gorm.DB) (models.AgentWorkflow, error) {
	var wfr models.AgentWorkFloNodeUpdateRequest
	wfr.NodeID = req.NodeID
	wfr.AgentId = req.AgentId
	wfr.OrgId = req.OrgId
	wfr.WorkflowId = req.WorkflowId
	wfr.Config = req.Config

	return wfr.UpdateWorkflowNode(db)
}
