package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

type WorkFlow struct {
	ID              string                `gorm:"type:uuid;primaryKey" json:"id"`
	UserId          string                `gorm:"type:uuid" json:"-"`
	OrgId           string                `gorm:"type:uuid" json:"-"`
	Name            string                `gorm:"type:text" json:"name"`
	Description     string                `gorm:"type:text" json:"description"`
	Tags            []string              `gorm:"type:text[]" json:"tags"`
	Meta            json.RawMessage       `gorm:"type:jsonb" json:"meta"`
	Agents          []string              `gorm:"type:text[]" json:"agents"`
	FlowConnections Connections           `gorm:"type:jsonb" json:"connections"`
	Settings        WorkflowSettingsEntry `gorm:"type:jsonb" json:"settings"`
	CreatedAt       time.Time             `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	UpdatedAt       time.Time             `gorm:"type:timestamp;default:current_timestamp" json:"-"`
}

type WorkFlowRequest struct {
	UserId          string                `json:"-"`
	OrgId           string                `json:"-"`
	Id              string                `json:"id"`
	Name            string                `json:"name"`
	Description     string                `json:"description"`
	Tags            []string              `json:"tags"`
	Meta            json.RawMessage       `json:"meta"`
	Agents          []string              `json:"agents"`
	FlowConnections Connections           `json:"connections"`
	Settings        WorkflowSettingsEntry `json:"settings"`
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
	MaxExecutionTime int    `json:"maxExecutionTime"`
	RetryPolicy      string `json:"retryPolicy"`
	ErrorHandling    string `json:"errorHandling"`
}

type Connections []Connections
type WorkflowSettingsEntry []WorkflowSettings

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

func (wf *WorkFlow) CreateWorkflow(db *gorm.DB) error {
	var existing WorkFlow
	err := db.First(&existing, "id = ? AND user_id = ? AND org_id = ?", wf.ID, wf.UserId, wf.OrgId).Error

	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&wf).Error
	}

	return err
}

func (wf *WorkFlow) UpdateWorkflow(db *gorm.DB) error {
	return db.Model(&WorkFlow{}).
		Where("id = ? AND user_id = ? AND org_id = ?", wf.ID, wf.UserId, wf.OrgId).
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
	return db.Where("id = ? AND user_id = ?", req.Id, req.UserId).Delete(&WorkFlow{}).Error
}

func ListWorkflows(db *gorm.DB, req WorkFlowRequest) ([]WorkflowSummary, error) {
	var wfs []WorkflowSummary
	err := db.Where("org_id = ?", req.OrgId).Find(&wfs).Error
	return wfs, err
}

func GetWorkflowByID(db *gorm.DB, req WorkFlowRequest) (WorkFlow, error) {
	var wf WorkFlow
	err := db.Where("id = ?  AND org_id = ?", req.Id, req.UserId, req.OrgId).First(&wf).Error
	return wf, err
}
