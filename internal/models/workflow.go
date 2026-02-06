package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

type Workflow struct {
	ID              string                `gorm:"type:uuid;primaryKey" json:"id"`
	UserId          string                `gorm:"type:uuid" json:"-"`
	OrgId           string                `gorm:"type:uuid" json:"-"`
	Name            string                `gorm:"type:text" json:"name"`
	Description     string                `gorm:"type:text" json:"description"`
	Tags            StringSlice           `gorm:"type:jsonb" json:"tags"`
	Meta            JSONBMap              `gorm:"type:jsonb" json:"meta"`
	RawEntry        JSONBMap              `gorm:"type:jsonb" json:"raw_entry"`
	Agents          StringSlice           `gorm:"type:jsonb" json:"agents_id"`
	FlowConnections Connections           `gorm:"type:jsonb" json:"connections"`
	Settings        WorkflowSettingsEntry `gorm:"type:jsonb" json:"settings"`
	Category        string                `gorm:"type:text;default:default" json:"category"`
	CreatedAt       time.Time             `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	UpdatedAt       time.Time             `gorm:"type:timestamp;default:current_timestamp" json:"-"`
}

type GeneralWorkflow struct {
	ID               string                `gorm:"type:uuid;primaryKey" json:"id"`
	Name             string                `gorm:"type:text" json:"name"`
	Description      string                `gorm:"type:text" json:"description"`
	Tags             StringSlice           `gorm:"type:jsonb" json:"tags"`
	Meta             JSONBMap              `gorm:"type:jsonb" json:"meta"`
	RawEntry         JSONBMap              `gorm:"type:jsonb" json:"raw_entry"`
	Agents           StringSlice           `gorm:"type:jsonb" json:"agents_id"`
	FlowConnections  Connections           `gorm:"type:jsonb" json:"connections"`
	Settings         WorkflowSettingsEntry `gorm:"type:jsonb" json:"settings"`
	Category         string                `gorm:"type:text;default:default" json:"category"`
	CreatedAt        time.Time             `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	UpdatedAt        time.Time             `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	ShortDescription string                `gorm:"type:text" json:"short_description"`
	LongDescription  string                `gorm:"type:text" json:"long_description"`
}

type WorkFlowRequest struct {
	UserId          string                `json:"-"`
	OrgId           string                `json:"-"`
	Id              string                `json:"id"`
	Name            string                `json:"name" validate:"required"`
	Description     string                `json:"description" validate:"required"`
	Tags            StringSlice           `json:"tags"`
	Meta            JSONBMap              `json:"meta"`
	Agents          StringSlice           `json:"agents_id" validate:"required,dive,uuid"`
	FlowConnections Connections           `json:"connections"`
	Settings        WorkflowSettingsEntry `json:"settings"`
	Category        string                `json:"category"`
}

type ChannelWorkflow struct {
	ID         string    `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ChannelID  string    `json:"channel_id" gorm:"type:uuid;not null"`
	WorkflowID string    `json:"workflow_id" gorm:"type:uuid;not null"`
	CreatedAt  time.Time `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	UpdatedAt  time.Time `gorm:"type:timestamp;default:current_timestamp" json:"-"`
}

type AgentWorkflow struct {
	ID         string `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AgentId    string `json:"agent_id" gorm:"type:uuid;not null"`
	WorkflowId string `json:"workflow_id" gorm:"type:uuid;not null"`
	UserID     string `json:"user_id" gorm:"type:uuid"`
	// Private          bool      `gorm:"type:boolean;default:true" json:"private"`
	RawEntry         JSONBMap  `gorm:"type:jsonb" json:"raw_entry"`
	Name             string    `gorm:"type:text" json:"name"`
	OrgId            string    `gorm:"type:uuid" json:"-"`
	IsActive         bool      `gorm:"type:boolean" json:"is_active"`
	Category         string    `gorm:"type:text" json:"category"`
	CreatedAt        time.Time `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	UpdatedAt        time.Time `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	ShortDescription string    `gorm:"type:text" json:"short_description"`
	LongDescription  string    `gorm:"type:text" json:"long_description"`
	Description      string    `gorm:"type:text" json:"description"`
	IsPublic         bool      `gorm:"type:boolean;default:false" json:"-"`
}

type AgentWorkFlowRequest struct {
	RawEntry         JSONBMap `json:"raw_entry" validate:"required"`
	AgentId          string   `json:"-"`
	Name             string   `json:"name" validate:"required"`
	OrgId            string   `json:"-"`
	UserID           string   `json:"-"`
	WorkflowId       string   `json:"-"`
	ShortDescription string   `json:"short_description"`
	LongDescription  string   `json:"long_description"`
	Description      string   `json:"description"`
	Category         string   `json:"category"`
	IsPublic         bool     `json:"-"`
}

type AgentWorkFloUpdateRequest struct {
	RawEntry   JSONBMap `json:"raw_entry" validate:"required"`
	AgentId    string   `json:"-"`
	IsActive   bool     `json:"is_active"`
	Name       string   `json:"name"`
	Private    bool     `json:"private"`
	OrgId      string   `json:"-"`
	WorkflowId string   `json:"-"`
}

type AgentWorkFloNodeUpdateRequest struct {
	RawEntry   JSONBMap    `json:"raw_entry"`
	AgentId    string      `json:"agent_id" validate:"required"`
	IsActive   bool        `json:"is_active"`
	Name       string      `json:"name"`
	OrgId      string      `json:"-"`
	WorkflowId string      `json:"-"`
	NodeID     string      `json:"node_id"`
	NodeType   string      `json:"node_type"`
	Config     JSONBMapArr `json:"config"`
}
type ChannelWorkflowRequest struct {
	ChannelID  string `json:"channel_id"`
	WorkflowID string `json:"workflow_id"`
}

type AgentsDetails struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	URL          string `json:"url"`
	Name         string `json:"name"`
	ResponseMode string `json:"responseMode"`
	MaxRetries   int    `json:"maxRetries"`
	Timeout      int    `json:"timeout"`
}

type WorkFlowResponse struct {
	Workflow
	Agents []AgentsDetails `json:"agents_details"`
}

type AgentWorkFlowResponse struct {
	AgentId          string    `json:"agent_id"`
	WorkflowId       string    `json:"workflow_id"`
	RawEntry         JSONBMap  `json:"raw_entry"`
	Name             string    `json:"name"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	ShortDescription string    `json:"short_description"`
	LongDescription  string    `json:"long_description"`
	Description      string    `json:"description"`
}

type WorkflowSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ISActive bool   `json:"is_active"`
}

type AgentWorkflowSummary struct {
	WorkflowId       string   `json:"workflow_id"`
	Name             string   `json:"name"`
	RawEntry         JSONBMap `json:"raw_entry"`
	IsActive         bool     `json:"is_active"`
	ShortDescription string   `json:"short_description"`
	LongDescription  string   `json:"long_description"`
	Description      string   `json:"description"`
}

type Connection struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition"`
}

type WorkflowSettings struct {
	MaxExecutionTime int    `json:"max_execution_time"`
	RetryPolicy      string `json:"retry_policy"`
	ErrorHandling    string `json:"error_handling"`
}

type Connections []Connection
type WorkflowSettingsEntry []WorkflowSettings
type StringSlice []string
type JSONBMap map[string]interface{}

func (s StringSlice) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *StringSlice) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("StringSlice: Scan source is not []byte")
	}
	return json.Unmarshal(bytes, s)
}

func (j *JSONBMap) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}

func (j JSONBMap) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (c Connections) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *Connections) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for Connection")
	}
	return json.Unmarshal(bytes, c)
}

func (s WorkflowSettingsEntry) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *WorkflowSettingsEntry) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for WorkflowSettings")
	}
	return json.Unmarshal(bytes, s)
}

func (AgentWorkflowSummary) TableName() string {
	return "agent_workflows"
}

func (wf *Workflow) CreateWorkflow(db *gorm.DB) error {

	if err := ValidateAgentIDs(db, wf.OrgId, wf.Agents); err != nil {
		return err
	}

	return db.Create(&wf).Error

}

func (wf *AgentWorkflow) CreateAgentWorkflow(db *gorm.DB) (error, int) {
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error, http.StatusInternalServerError
	}

	if err := ValidateAgentIDs(tx, wf.OrgId, []string{wf.AgentId}); err != nil {
		tx.Rollback()
		return err, http.StatusBadRequest
	}

	var existing AgentWorkflow
	err := tx.Where("org_id = ? AND agent_id = ? AND user_id = ?",
		wf.OrgId, wf.AgentId, wf.UserID).First(&existing).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			wf.WorkflowId = utility.GenerateUUID()
			if err := tx.Create(&wf).Error; err != nil {
				tx.Rollback()
				return err, http.StatusInternalServerError
			}
		} else {
			tx.Rollback()
			return err, http.StatusInternalServerError
		}
	} else {
		if err := tx.Unscoped().
			Where("workflow_id = ?", existing.WorkflowId).
			Delete(&WorkflowNode{}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete existing workflow nodes: %w", err),
				http.StatusInternalServerError
		}

		existing.RawEntry = wf.RawEntry
		existing.Name = wf.Name

		if err := tx.Save(&existing).Error; err != nil {
			tx.Rollback()
			return err, http.StatusInternalServerError
		}

		wf.WorkflowId = existing.WorkflowId
		wf.ID = existing.ID
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err),
			http.StatusInternalServerError
	}

	return nil, http.StatusOK
}

func (wf *Workflow) UpdateWorkflow(db *gorm.DB) error {

	if err := ValidateAgentIDs(db, wf.OrgId, wf.Agents); err != nil {
		return err
	}

	return db.Model(&Workflow{}).
		Where("id = ? AND org_id = ?", wf.ID, wf.OrgId).
		Updates(map[string]interface{}{
			"name":             wf.Name,
			"description":      wf.Description,
			"tags":             wf.Tags,
			"meta":             wf.Meta,
			"agents":           wf.Agents,
			"flow_connections": wf.FlowConnections,
			"settings":         wf.Settings,
		}).Error
}

func (wf *AgentWorkflow) UpdateAgentWorkflow(db *gorm.DB) (error, int) {

	if err := ValidateAgentIDs(db, wf.OrgId, []string{wf.AgentId}); err != nil {
		return err, http.StatusBadRequest
	}

	err := db.Model(&AgentWorkflow{}).
		Where("workflow_id = ? AND org_id = ? AND agent_id = ?", wf.WorkflowId, wf.OrgId, wf.AgentId).
		Updates(map[string]interface{}{
			"raw_entry": wf.RawEntry,
			"is_active": wf.IsActive,
			"name":      wf.Name,
		}).Error
	if err != nil {
		return err, http.StatusInternalServerError
	}

	return nil, http.StatusOK
}

func (n *AgentWorkFloNodeUpdateRequest) UpdateWorkflowNode(db *gorm.DB) (AgentWorkflow, error) {
	wfr := AgentWorkflow{}

	exists := postgresql.CheckExists(db, &wfr, "workflow_id = ? AND agent_id = ? AND org_id = ?", n.WorkflowId, n.AgentId, n.OrgId)

	if !exists {
		return wfr, errors.New("agent not attached to a workflow")
	}

	parameters := JSONBMapArr{JSONBMap{}}

	//The current algorithm for converting config to parameters.
	for _, v := range n.Config {
		var valueParam interface{}

		if value, ok := v["value"]; ok {
			valueParam = value
		} else {
			valueParam = v["default"]
		}

		con := parameters[0]
		con[v["name"].(string)] = valueParam
		parameters[0] = con
	}

	// Update the parameters matching the skill id in the workflow
	rawEntry := wfr.RawEntry

	nodes, ok := rawEntry["nodes"].([]interface{})
	if !ok {
		return wfr, errors.New("workflow has no nodes")
	}

	for i, node := range nodes {
		nodeMap, ok := node.(map[string]interface{})
		if !ok {
			continue
		}

		if nodeMap["id"] == n.NodeID {

			nodeMap["parameters"] = parameters

			nodes[i] = nodeMap
			rawEntry["nodes"] = nodes

			break
		}
		// add something for node type later
	}

	wfUpdates := map[string]any{
		"raw_entry": rawEntry,
	}

	result, err := postgresql.UpdateFields(db, &wfr, wfUpdates, "workflow_id = ? AND agent_id = ?", wfr.WorkflowId, wfr.AgentId)

	if err != nil {
		return wfr, errors.New("failed to update agent workflow")
	}

	if result.RowsAffected == 0 {
		return wfr, errors.New("no record updated")
	}

	return wfr, nil
}

func (wf *AgentWorkflow) DeleteWorkflow(db *gorm.DB) (error, int) {
	if err := ValidateAgentIDs(db, wf.OrgId, []string{wf.AgentId}); err != nil {
		return err, http.StatusBadRequest
	}

	err := db.Where("workflow_id = ? AND org_id = ? AND agent_id = ?", wf.WorkflowId, wf.OrgId, wf.AgentId).Delete(&AgentWorkflow{}).Error

	if err != nil {
		return err, http.StatusInternalServerError
	}

	return nil, http.StatusOK
}

func DeleteWorkflow(db *gorm.DB, req WorkFlowRequest) error {
	return db.Where("id = ?", req.Id).Delete(&Workflow{}).Error
}

func ListWorkflows(db *gorm.DB, req WorkFlowRequest) ([]WorkflowSummary, error) {
	wfs := []WorkflowSummary{}
	err := db.Table("workflows").Where("org_id = ?", req.OrgId).Scan(&wfs).Error
	return wfs, err
}

func (wf *AgentWorkflow) ListWorkflows(db *gorm.DB, c *gin.Context) ([]AgentWorkflowSummary, postgresql.PaginationResponse, int, error) {
	var wfs []AgentWorkflowSummary

	var lastQuery string
	var params []any

	if wf.IsPublic {
		lastQuery = "(agent_workflows.agent_id::text = ? OR agent_workflows.agent_id::text LIKE ?) AND agent_workflows.is_public = ?"
		params = []any{wf.AgentId, "%-" + wf.AgentId, true}
	} else {
		lastQuery = "agent_workflows.agent_id = ? AND agent_workflows.org_id = ?"
		params = []any{wf.AgentId, wf.OrgId}
	}

	pagination := postgresql.GetPagination(c)

	query := db.Model(&AgentWorkflow{}).
		Select(`
			agent_workflows.workflow_id,
			agent_workflows.agent_id,
			agent_workflows.raw_entry,
			agent_workflows.is_active,
			agent_workflows.created_at,
			COALESCE(general_workflows.name, agent_workflows.name) AS name,
			COALESCE(general_workflows.description, agent_workflows.description) AS description,
			COALESCE(general_workflows.tags, '[]') AS tags,
			COALESCE(general_workflows.category, agent_workflows.category) AS category,
			COALESCE(general_workflows.short_description, agent_workflows.short_description) AS short_description,
			COALESCE(general_workflows.long_description, agent_workflows.long_description) AS long_description
		`).
		Joins("LEFT JOIN general_workflows ON general_workflows.id = agent_workflows.workflow_id").
		Where(lastQuery, params...)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&wfs,
		nil,
	)
	if err != nil {
		return wfs, paginationResponse, http.StatusInternalServerError, err
	}

	return wfs, paginationResponse, http.StatusOK, nil
}

func GetWorkflowByID(db *gorm.DB, req WorkFlowRequest) (WorkFlowResponse, error) {
	wf := Workflow{}
	err := db.Where("id = ? AND org_id = ?", req.Id, req.OrgId).First(&wf).Error
	agentDetails, err := FetchAgentsFromIntegration(db, wf.Agents)
	if err != nil {
		return WorkFlowResponse{}, err
	}

	resp := WorkFlowResponse{
		wf,
		agentDetails,
	}
	return resp, err
}

func (wf *AgentWorkflow) GetWorkflowByID(db *gorm.DB) (*AgentWorkFlowResponse, int, error) {
	wfr := AgentWorkFlowResponse{}

	if err := ValidateAgentIDs(db, wf.OrgId, []string{wf.AgentId}); err != nil {
		return &wfr, http.StatusBadRequest, err
	}

	err := db.Model(&AgentWorkflow{}).Where("agent_id = ? AND org_id = ? AND workflow_id = ?", wf.AgentId, wf.OrgId, wf.WorkflowId).Scan(&wfr).Error
	if err != nil {
		return &wfr, http.StatusInternalServerError, err
	}

	return &wfr, http.StatusOK, err
}

func ValidateAgentIDs(db *gorm.DB, orgID string, agentIDs []string) error {
	var validIDs []string
	if err := db.Model(&OrganisationIntegrations{}).
		Where("integration_id IN ?", agentIDs).
		Pluck("integration_id", &validIDs).Error; err != nil {
		return fmt.Errorf("error validating agents: %w", err)
	}

	validMap := make(map[string]bool)
	for _, id := range validIDs {
		validMap[id] = true
	}

	var invalid []string
	for _, id := range agentIDs {
		if !validMap[id] {
			invalid = append(invalid, id)
		}
	}

	if len(invalid) > 0 {
		return fmt.Errorf("invalid agent IDs: %v, agents does not exist in Org", invalid)
	}

	return nil
}

func FetchAgentsFromIntegration(db *gorm.DB, agentIDs []string) ([]AgentsDetails, error) {
	var integrations []OrganisationIntegrations

	if err := db.
		Where("integration_id IN ?", agentIDs).
		Find(&integrations).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch agents: %w", err)
	}

	// Create a map for quick lookup to preserve order
	integrationMap := make(map[string]OrganisationIntegrations)
	for _, integration := range integrations {
		integrationMap[integration.IntegrationID] = integration
	}

	var agents []AgentsDetails
	for _, id := range agentIDs {
		if integ, ok := integrationMap[id]; ok {
			agent := AgentsDetails{
				ID:           integ.ID,
				Type:         "agent",
				URL:          integ.AppUrl,
				Name:         integ.AppName,
				ResponseMode: "stream",
				MaxRetries:   3,
				Timeout:      60,
			}
			agents = append(agents, agent)
		}
	}

	return agents, nil
}

func (w *Workflow) CheckWorkflowExists(db *gorm.DB, workflowID string) (bool, error) {
	exists := postgresql.CheckExists(db, &w, "id = ?", workflowID)
	if !exists {
		return exists, errors.New("workflow does not exist")
	}
	return exists, nil
}

func (w *GeneralWorkflow) CheckWorkflowExists(db *gorm.DB, workflowID string) (bool, error) {
	exists := postgresql.CheckExists(db, &w, "id = ?", workflowID)
	if !exists {
		return exists, errors.New("workflow does not exist")
	}
	return exists, nil
}

func (wc *ChannelWorkflow) CheckChannelWorkflowExists(db *gorm.DB) (bool, error) {
	exists := postgresql.CheckExists(db, &wc, "channel_id = ? AND workflow_id = ?", wc.ChannelID, wc.WorkflowID)
	if !exists {
		return exists, errors.New("channel workflow does not exist")
	}
	return exists, nil
}

func (cw *ChannelWorkflow) Add(db *gorm.DB) (int, error) {

	var channel Channels
	channelExists, err := channel.CheckChannelExists(db, cw.ChannelID)
	if err != nil || !channelExists {
		return http.StatusUnprocessableEntity, fmt.Errorf("channel does not exist: %v", err)
	}

	var workflow Workflow
	workflowExists, err := workflow.CheckWorkflowExists(db, cw.WorkflowID)
	if err != nil || !workflowExists {
		return http.StatusUnprocessableEntity, fmt.Errorf("workflow does not exist: %v", err)
	}

	if err := db.Create(cw).Error; err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusCreated, nil
}

func (cw *ChannelWorkflow) RemoveChannelWorkflow(db *gorm.DB) (int, error) {

	exist, err := cw.CheckChannelWorkflowExists(db)
	if err != nil || !exist {
		return http.StatusUnprocessableEntity, fmt.Errorf("channel workflow does not exist: %v", err)
	}

	if err := db.Delete(&cw).Error; err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to remove workflow from channel: %v", err)
	}

	return http.StatusOK, nil
}

func (cw *ChannelWorkflow) GetWorkflowsByChannel(db *gorm.DB) ([]Workflow, error) {
	workflows := []Workflow{}
	err := db.Table("workflows").
		Joins("JOIN channel_workflows ON workflows.id = channel_workflows.workflow_id").
		Where("channel_workflows.channel_id = ?", cw.ChannelID).
		Scan(&workflows).Error
	return workflows, err
}

func (cw *GeneralWorkflow) GetMarketPlaceWorkflowById(db *gorm.DB) (*[]GeneralWorkflow, error) {
	workflows := []GeneralWorkflow{}
	err := db.Table("general_workflows").
		Where("general_workflow.id = ?", cw.ID).
		Scan(&workflows).Error
	return &workflows, err
}

func (cw *GeneralWorkflow) GetMarketPlaceWorkflows(db *gorm.DB, c *gin.Context) (*[]GeneralWorkflow, postgresql.PaginationResponse, error) {
	workflows := []GeneralWorkflow{}
	pagination := postgresql.GetPagination(c)
	query := db.Table("general_workflows")

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&workflows,
		nil,
	)

	return &workflows, paginationResponse, err
}

func (cw *ChannelWorkflow) GetWorkflowsWithChannelStatus(db *gorm.DB, orgId *string) ([]WorkflowSummary, error) {
	var results []WorkflowSummary

	subQuery := db.Model(&ChannelWorkflow{}).
		Select("1").
		Where("channel_workflows.workflow_id = workflows.id").
		Where("channel_workflows.channel_id = ?", cw.ChannelID)

	err := db.Model(&Workflow{}).
		Select("workflows.id, workflows.name, EXISTS (?) AS is_active", subQuery).
		Where("workflows.org_id = ?", *orgId).
		Scan(&results).Error

	return results, err
}

func (w *ChannelWorkflow) DeleteChannelWorkflows(db *gorm.DB) error {
	return postgresql.DeleteSpecificRecord(db, &ChannelWorkflow{}, "channel_id = ?", w.ChannelID)
}
