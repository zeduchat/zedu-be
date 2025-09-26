package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

type NodeConnections [][]NodeConnection
type ProcessSteps []ProcessStep
type StepRequests []StepReq

type Prompts struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null"`
	Template  string    `json:"template" gorm:"type:text;not null"`
	Version   int       `json:"version" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type ProcessStep struct {
	Step    string `json:"step"`
	Input   string `json:"input"`
	Output  string `json:"output"`
	Status  string `json:"status"`
	Prompt  string `json:"prompt"`
	LLMCall bool   `json:"llm_call"`
}

type TranslationRequest struct {
	TaskList string       `json:"task_list" binding:"required"`
	Skills   []string     `json:"skills" binding:"required"`
	Steps    StepRequests `json:"steps" binding:"required"`
}

type StepReq struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
}

type MissingSkillsResponse struct {
	MissingSkills []string `json:"missing_skills"`
	Suggestion    string   `json:"suggestion"`
}
type WorkflowJSON struct {
	ID          string                   `json:"id"`
	Meta        WorkflowMeta             `json:"meta"`
	Name        string                   `json:"name"`
	Tags        []string                 `json:"tags"`
	Nodes       []Node                   `json:"nodes"`
	Active      bool                     `json:"active"`
	PinData     map[string]any           `json:"pinData"`
	Settings    WkfSettings              `json:"settings"`
	VersionID   string                   `json:"versionId"`
	Connections map[string]NodeConnector `json:"connections"`
}

type WorkflowMeta struct {
	InstanceID                  string `json:"instanceId"`
	TemplateCredsSetupCompleted bool   `json:"templateCredsSetupCompleted"`
}

type WkfSettings struct {
	ExecutionOrder string `json:"executionOrder"`
}

type Node struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	NodeName    string         `json:"node_name"`
	Type        string         `json:"type"`
	SkillID     string         `json:"skill_id"`
	Position    []int          `json:"position"`
	Params      map[string]any `json:"parameters"`
	TypeVersion float64        `json:"typeVersion"`
	WebhookID   string         `json:"webhookId,omitempty"`
}

type NodeConnector struct {
	Main NodeConnections `json:"main"` // Connections between nodes
}

type NodeConnection struct {
	Node  string `json:"node"`  // Destination node name
	Type  string `json:"type"`  // Connection type (usually "main")
	Index int    `json:"index"` // Connection index
}

type TranslationResponse struct {
	Status string `json:"status"` //success, failed, incomplete
	// Workflow   map[string]any `json:"workflow,omitempty"`
	ProcessStep   ProcessSteps           `json:"process_step,omitempty"`
	MissingSkills *MissingSkillsResponse `json:"missing_skills,omitempty"`
}

type PromptVersionDetail struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	Template  string    `json:"template"`
	CreatedAt time.Time `json:"created_at"`
}

type PromptVersionList []PromptVersionDetail

// Implement sql.Scanner
func (p *PromptVersionList) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan PromptVersionList: not []byte")
	}
	return json.Unmarshal(bytes, p)
}

// (Optional) Implement driver.Valuer if you ever write back
func (p PromptVersionList) Value() (driver.Value, error) {
	return json.Marshal(p)
}

type GetPromptResponse struct {
	PromptName string            `json:"prompt_name"`
	Versions   PromptVersionList `json:"versions"`
}

func (p *Prompts) BeforeCreate(tx *gorm.DB) (err error) {
	var maxVersion int
	p.ID = utility.GenerateUUID()

	err = tx.Model(&Prompts{}).
		Where("name = ?", p.Name).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion).Error
	if err != nil {
		return err
	}

	p.Version = maxVersion + 1
	return nil
}

func (p *Prompts) CreatePrompt(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &p)
	if err != nil {
		return fmt.Errorf("unable to create prompt: %v", err)
	}

	return nil
}

func (p *Prompts) GetAllPrompts(db *gorm.DB) ([]Prompts, int, error) {
	var prompts []Prompts

	err := postgresql.SelectAllFromDb(db, "", &prompts, "")
	if err != nil {
		return prompts, http.StatusBadRequest, fmt.Errorf("unable to fetch all prompts: %v", err)
	}

	return prompts, http.StatusOK, nil
}

func (p *Prompts) GetPrompt(db *gorm.DB, name string) (GetPromptResponse, int, error) {
	query := `
		SELECT 
			name AS prompt_name,
			json_agg(
				json_build_object(
					'id', id,
					'template', template,
					'created_at', created_at,
					'version', version
				) ORDER BY version DESC
			) AS versions
		FROM prompts
		WHERE name = ?
		GROUP BY name;
	`

	var resp GetPromptResponse
	if err := db.Raw(query, name).Scan(&resp).Error; err != nil {
		return resp, http.StatusBadRequest, err
	}

	return resp, http.StatusOK, nil
}

func (p *Prompts) GetPromptByVersion(db *gorm.DB, req StepReq) (int, error) {
	err := db.
		Where("name = ? AND version = ?", req.Name, req.Version).
		First(&p).Error

	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("unable to fetch prompt: %v", err)
	}

	return http.StatusOK, nil
}

func (p *Prompts) GetLatestPromptVersionByName(db *gorm.DB, name string) error {
	err := db.
		Where("name = ?", name).
		Order("version DESC").
		First(&p).Error

	if err != nil {
		return fmt.Errorf("unable to fetch latest prompt version: %v", err)
	}

	return nil
}

func (p *Prompts) FetchUniquePrompts(db *gorm.DB) ([]Prompts, error) {
	var prompts []Prompts

	subQuery := db.Model(&Prompts{}).
		Select("name, MAX(created_at) as max_created_at").
		Group("name")

	err := db.Model(&Prompts{}).
		Joins("JOIN (?) as latest ON prompts.name = latest.name AND prompts.created_at = latest.max_created_at", subQuery).
		Find(&prompts).Error

	if err != nil {
		return nil, err
	}

	return prompts, nil
}
