package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Workflow struct {
	ID              string                `gorm:"type:uuid;primaryKey" json:"id"`
	UserId          string                `gorm:"type:uuid" json:"-"`
	OrgId           string                `gorm:"type:uuid" json:"-"`
	Name            string                `gorm:"type:text" json:"name"`
	Description     string                `gorm:"type:text" json:"description"`
	Tags            StringSlice           `gorm:"type:jsonb" json:"tags"`
	Meta            JSONBMap              `gorm:"type:jsonb" json:"meta"`
	Agents          StringSlice           `gorm:"type:jsonb" json:"agents_id"`
	FlowConnections Connections           `gorm:"type:jsonb" json:"connections"`
	Settings        WorkflowSettingsEntry `gorm:"type:jsonb" json:"settings"`
	CreatedAt       time.Time             `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	UpdatedAt       time.Time             `gorm:"type:timestamp;default:current_timestamp" json:"-"`
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

type WorkflowSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

func (wf *Workflow) CreateWorkflow(db *gorm.DB) error {

	if err := ValidateAgentIDs(db, wf.OrgId, wf.Agents); err != nil {
		return err
	}

	return db.Create(&wf).Error

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

func DeleteWorkflow(db *gorm.DB, req WorkFlowRequest) error {
	return db.Where("id = ?", req.Id).Delete(&Workflow{}).Error
}

func ListWorkflows(db *gorm.DB, req WorkFlowRequest) ([]WorkflowSummary, error) {
	var wfs []WorkflowSummary
	err := db.Table("workflows").Where("org_id = ?", req.OrgId).Scan(&wfs).Error
	return wfs, err
}

func GetWorkflowByID(db *gorm.DB, req WorkFlowRequest) (WorkFlowResponse, error) {
	var wf Workflow
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

func ValidateAgentIDs(db *gorm.DB, orgID string, agentIDs []string) error {
	var validIDs []string

	if err := db.Model(&OrganisationIntegrations{}).
		Where("org_id = ? AND integration_id IN ?", orgID, agentIDs).
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
