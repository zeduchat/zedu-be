package models

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

type Integrations struct {
	ID                 string             `gorm:"type:uuid;primary_key" json:"id"`
	Name               string             `gorm:"colume:name; type:varchar(255); not null;unique" json:"name"`
	JSONUrl            string             `gorm:"column:json_url; type:varchar(255);" json:"-"`
	AppUrl             string             `gorm:"column:app_url; type:varchar(255);" json:"app_url"`
	AppLogo            string             `gorm:"column:app_logo; type:varchar(255);" json:"avatar"`
	OwnerID            string             `gorm:"type:uuid;" json:"owner_id"`
	AppDescription     string             `gorm:"column:app_description; type:varchar(255);" json:"description"`
	IntegrationType    string             `gorm:"column:integration_type; type:varchar(255);" json:"-"`
	Info               string             `gorm:"colummn:info; type:varchar(255);" json:"-"`
	IsActive           bool               `gorm:"type:boolean;default:false" json:"is_active"`
	IsPaid             bool               `gorm:"type:boolean;default:false" json:"-"`
	IsApproved         bool               `gorm:"type:boolean;default:false" json:"-"`
	Prices             JSONPrices         `gorm:"type:jsonb" json:"prices"`
	Version            string             `gorm:"type:varchar(20);default:'v1.0.0'" json:"version"`
	Provider           Provider           `gorm:"type:jsonb" json:"provider"`
	DefaultInputModes  []string           `gorm:"type:jsonb" json:"default_input_modes"`
	DefaultOutputModes []string           `gorm:"type:jsonb" json:"default_output_modes"`
	PreSharedKey       string             `gorm:"type:varchar(64);uniqueIndex" json:"preshared_key"`
	CreatedAt          time.Time          `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time          `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	Skills             JSONSkills         `gorm:"type:jsonb" json:"skills"`
	IsSystem           bool               `gorm:"type:boolean;default:false" json:"is_system"`
	CommissionRate     float64            `gorm:"type:decimal(5,2);default:80.00" json:"commission_rate"` // default commission rate for agent bill is 80% and telex takes 20% -> 0.8/0.2
	Capabilities       CapabilitiesObject `gorm:"type:jsonb" json:"capabilities"`
	Tone               string             `gorm:"column:tone;type:varchar(255);default:friendly" json:"tone"`
	Title              string             `gorm:"column:title;type:text;" json:"title"`
	Visibility         string             `gorm:"column:title;type:varchar(255)" json:"visibility"`
	SystemPrompts      JSONSystemPrompts  `gorm:"type:jsonb" json:"system_prompts"`
	Category           string             `gorm:"type:text" json:"category"`
}

type CreateAgentRequest struct {
	Name          string            `json:"name" validate:"required"`
	Tone          string            `json:"tone" validate:"required"`
	Avatar        string            `json:"avatar"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Visibility    string            `json:"visibility" validate:"required,oneof=private public me"`
	SystemPrompts JSONSystemPrompts `json:"system_prompts"`
	UserId        string
	OrgId         string
	AgentId       string
}

type UpdateAgentPromptRequest struct {
	SystemPrompts JSONSystemPrompts `json:"system_prompts" validate:"required"`
	AgentId       string
	UserId        string
	OrgId         string
}

type UpdateAgent struct {
	Name            string `json:"name"`
	JSONUrl         string `json:"json_url"`
	AuthCredential  string `json:"auth_credential"`
	IntegrationType string `json:"integration_type"`
}

type AdminUpdateAgent struct {
	IsActive   bool `json:"is_active"`
	IsApproved bool `json:"is_approved"`
	IsSystem   bool `json:"is_system"`
}

type ChangeAgentStatus struct {
	Status     bool   `json:"status" validate:"required,oneof=true false"`
	AgentID    string `json:"integration_id"`
	JSONSchema JSONB  `gorm:"column:json_schema; type:jsonb;serializer:json" json:"json_schema"`
}

type OutputIntegrationsResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ChannelsUrl string `json:"channels_url"`
}

type UpdateJSONSchemaRequest struct {
	JSONSchema JSONB `gorm:"column:json_schema; type:jsonb;serializer:json" json:"json_schema"`
}

type CustomIntegrationRequest struct {
	JSONUrl        string `json:"json_url" validate:"required"`
	AppName        string
	AppLogo        string
	AppUrl         string
	AppDescription string
}

type CustomIntegrationSettingRequest struct {
	SettingEntry    map[string]any `json:"setting_entry" validate:"required"`
	SerializedEntry string         `json:"serialized_entry"`
}

type ActivateChannelAgent struct {
	Status bool `json:"status"`
}

type CustomIntegrationsMetrics struct {
	All           int64   `json:"all"`
	Active        int64   `json:"active"`
	Inactive      int64   `json:"inactive"`
	Organizations int64   `json:"organizations"`
	Credits       float64 `json:"credits"`
}

type Price struct {
	Amount        float64 `json:"amount"`
	OperationType string  `json:"operation_type"`
	Currency      string  `json:"currency"`
}

type Provider struct {
	Organization string `json:"organization"`
	URL          string `json:"url"`
}

type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	InputModes  []string `json:"inputModes"`
	OutputModes []string `json:"outputModes"`
	Examples    []string `json:"exmaples"`
	Tags        []string `json:"tags"`
}

type SystemPrompts struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Type    string `json:"type"`
}

type CapabilitiesObject struct {
	Streaming              bool `json:"streaming"`
	PushNotifications      bool `json:"pushNotifications"`
	StateTransitionHistory bool `json:"stateTransitionHistory"`
}

type JSONPrices []Price
type JSONSkills []Skill
type JSONSystemPrompts []SystemPrompts

type OrganisationIntegrations struct {
	ID                 string             `gorm:"type:uuid;primary_key" json:"id"`
	OrgID              string             `gorm:"type:uuid;" json:"org_id"`
	IntegrationID      string             `gorm:"type:uuid;" json:"integration_id"`
	OwnerID            string             `gorm:"type:uuid;" json:"owner_id"`
	IsActive           bool               `gorm:"type:boolean;default:false" json:"is_active"`
	IsSystem           bool               `gorm:"type:boolean;default:false" json:"is_system"`
	IsArchived         bool               `gorm:"type:boolean;default:false" json:"is_archived"`
	ArchivedAt         time.Time          `gorm:"index" json:"-"`
	JSONSchema         JSONB              `gorm:"column:json_schema; type:jsonb;serializer:json" json:"-"`
	JSONUrl            string             `gorm:"type:text; column:json_url;" json:"json_url"`
	CreatedAt          time.Time          `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time          `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	AppDescription     string             `gorm:"column:app_description;type:text;" json:"app_description"`
	AppName            string             `gorm:"column:app_name;type:text;" json:"app_name"`
	AppLogo            string             `gorm:"column:app_logo;type:text;" json:"app_logo"`
	AppUrl             string             `gorm:"column:app_url; type:text;" json:"app_url"`
	IsPaid             bool               `gorm:"type:boolean;default:false" json:"is_paid"`
	IsApproved         bool               `gorm:"type:boolean;default:false" json:"is_approved"`
	Prices             JSONPrices         `gorm:"type:jsonb" json:"prices"`
	Version            string             `gorm:"type:varchar(20);default:'v1.0.0'" json:"version"`
	Provider           Provider           `gorm:"type:jsonb" json:"provider"`
	DefaultInputModes  []string           `gorm:"type:jsonb" json:"default_input_modes"`
	DefaultOutputModes []string           `gorm:"type:jsonb" json:"default_output_modes"`
	PreSharedKey       string             `gorm:"type:varchar(64)" json:"preshared_key"`
	Skills             JSONSkills         `gorm:"type:jsonb" json:"skills"`
	CommissionRate     float64            `gorm:"type:decimal(5,2);default:80.00" json:"commission_rate"` // default commission rate for agent bill is 80% and telex takes 20% -> 0.8/0.2
	Capabilities       CapabilitiesObject `gorm:"type:jsonb" json:"capabilities"`
	Tone               string             `gorm:"column:tone;type:varchar(255);default:friendly" json:"tone"`
	Title              string             `gorm:"column:title;type:text;" json:"title"`
	Visibility         string             `gorm:"column:visibility;type:varchar(255);default:public;" json:"visibility"`
	SystemPrompts      JSONSystemPrompts  `gorm:"type:jsonb" json:"system_prompts"`
}

type AdminAgentResp struct {
	Agent      Integrations `json:"agent"`
	User       User         `json:"user"`
	CreditUsed float64      `json:"credit_used"`
}

type CreditAggregate struct {
	IntegrationID string
	TotalUsed     float64
}

type AgentBillAggregate struct {
	IntegrationID     string
	MakerTotalEarning float64
	TelexTotalEarning float64
}

type AdminCustomAgentResp struct {
	Agent             OrganisationIntegrations `json:"agent"`
	User              User                     `json:"user"`
	CreditUsed        float64                  `json:"credit_used"`
	TelexTotalEarning float64                  `json:"telex_total_earning"`
	MakerTotalEarning float64                  `json:"maker_total_earning"`
}

type OrganisationChannelsIntegrations struct {
	ID            string    `gorm:"type:uuid;primary_key" json:"id"`
	OrgID         string    `gorm:"type:uuid;" json:"org_id"`
	IntegrationID string    `gorm:"type:uuid;" json:"integration_id"`
	ChannelID     string    `gorm:"type:uuid;" json:"channel_id"`
	IsActive      bool      `gorm:"type:boolean;default:false" json:"is_active"`
	IsSystem      bool      `gorm:"type:boolean;default:false" json:"is_system"`
	ArchivedAt    time.Time `gorm:"index" json:"-"`
	SendBack      bool      `gorm:"type:boolean;default:true" json:"send_back"`
	CreatedAt     time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

type IntegrationChansResp []struct {
	ChannelName string `json:"channel_name"`
	ChannelId   string `json:"channel_id"`
	IsActive    bool   `json:"is_active"`
}

type GetChannelIntResp []struct {
	Integrations
	SendBack bool `json:"send_back"`
}

type IntegrationOutput struct {
	ID                    string               `gorm:"type:uuid;primary_key" json:"id"`
	IntegrationModifierID string               `gorm:"type:uuid;" json:"integration_modifier_id"`
	IntegrationOutputID   string               `gorm:"type:uuid;" json:"integration_output_id"`
	IntegrationName       string               `gorm:"type:string;" json:"integration_name"`
	ChannelID             string               `gorm:"type:uuid;" json:"channel_id"`
	SendBack              bool                 `gorm:"type:boolean;" json:"send_back"`
	CreatedAt             time.Time            `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time            `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	IntegrationChannels   []IntegrationChannel `gorm:"foreignKey:IntegrationOutputID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"integration_channels"`
}

type PartialOrganisationIntegration struct {
	ID                string    `json:"id"`
	IntegrationID     *string   `json:"integration_id"`
	OrgID             *string   `json:"org_id"`
	IsActive          bool      `json:"is_active"`
	IsSystem          bool      `json:"is_system"`
	IsArchived        bool      `json:"is_archived"`
	JSONUrl           string    `json:"json_url"`
	AppName           string    `json:"app_name"`
	Name              string    `json:"name"`
	AppLogo           string    `json:"app_logo"`
	AppUrl            string    `json:"app_url"`
	IsPaid            bool      `json:"is_paid"`
	IsApproved        bool      `json:"is_approved"`
	CreatedAt         time.Time `json:"created_at"`
	Source            string    `json:"source"`
	Provider          Provider  `json:"provider"`
	CreditUsed        float64   `json:"credit_used"`
	TelexTotalEarning float64   `json:"telex_total_earning"`
	MakerTotalEarning float64   `json:"maker_total_earning"`
}

type Earnings struct {
	MakerTotal float64
	TelexTotal float64
}

type IntegrationChannel struct {
	ID                  string    `gorm:"type:uuid;primary_key" json:"id"`
	IntegrationOutputID string    `gorm:"type:uuid;" json:"-"`
	IntegrationID       string    `gorm:"type:uuid;" json:"integration_id"`
	OutputID            string    `gorm:"type:uuid;" json:"-"`
	ChannelID           string    `gorm:"type:uuid;" json:"channel_id"`
	IntChannelID        string    `gorm:"type:varchar(100);" json:"int_channel_id"`
	IntChannelName      string    `gorm:"type:varchar(100);" json:"int_channel_name"`
	CreatedAt           time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

type CustomIntegrationsSetting struct {
	ID            string    `gorm:"type:uuid;primary_key" json:"id"`
	OrgID         string    `gorm:"type:uuid;" json:"org_id"`
	IntegrationID string    `gorm:"type:uuid;" json:"integration_id"`
	IsSystem      bool      `gorm:"type:boolean;default:false" json:"is_system"`
	SettingEntry  string    `gorm:"type:text;" json:"setting_entry"`
	CreatedAt     time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

type AddIntegrationChannel struct {
	IntegrationModifierID string `json:"int_modifier_id" validate:"required"`
	IntegrationOutputID   string `json:"int_output_id" validate:"required"`
	ChannelID             string `json:"channel_id"`
	IntChannelID          string `json:"int_channel_id" validate:"required"`
	IntChannelName        string `json:"int_channel_name" validate:"required"`
}

type IntegrationChannelReq struct {
	ChannelID             string `json:"channel_id"`
	IntChannelID          string `json:"int_channel_id" validate:"required"`
	IntegrationModifierID string `json:"int_modifier_id" validate:"required"`
	IntegrationOutputID   string `json:"int_output_id" validate:"required"`
}

type AgentsResp []struct {
	Integrations
	Linked bool `json:"linked"`
}

type AgentResp struct {
	ID            string            `json:"id"`
	IsActive      bool              `json:"is_active"`
	Name          string            `json:"name"`
	Tone          string            `json:"tone"`
	Avatar        string            `json:"avatar"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Visibility    string            `json:"visibility"`
	SystemPrompts JSONSystemPrompts `json:"system_prompts"`
	Category      string            `json:"category,omitempty"`
}

type IntegrationBills struct {
	ID            string `gorm:"type:uuid;primary_key" json:"id"`
	OrgID         string `gorm:"type:uuid;" json:"org_id"`
	IntegrationID string `gorm:"type:uuid;" json:"integration_id"`
	MakerID       string `gorm:"type:uuid;" json:"maker_id"`
	CreditUsageID string `gorm:"type:uuid;" json:"credit_usage_id"`

	TelexAmount  float64 `gorm:"type:decimal(10,2);not null" json:"telex_amount"`
	MakerAmount  float64 `gorm:"type:decimal(10,2);not null" json:"maker_amount"`
	TotalAmount  float64 `gorm:"type:decimal(10,2);not null" json:"total_amount"`
	PayoutStatus string  `gorm:"type:varchar(20);default:'pending'" json:"payout_status"`

	CreatedAt time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`

	Organisation Organisation `gorm:"foreignKey:OrgID;references:ID"`
	User         User         `gorm:"foreignKey:MakerID;references:ID"`
}

type IntegrationBillsResponse struct {
	ID            string    `json:"id"`
	OrgID         string    `json:"org_id"`
	IntegrationID string    `json:"integration_id"`
	CreditUsageID string    `json:"credit_usage_id"`
	TelexAmount   float64   `json:"telex_amount"`
	MakerAmount   float64   `json:"maker_amount"`
	TotalAmount   float64   `json:"total_amount"`
	PayoutStatus  string    `json:"payout_status"`
	OrgName       string    `json:"org_name"`
	AppName       string    `json:"app_name"`
	OrgEmail      string    `json:"org_email"`
	MakerName     string    `json:"maker_name"`
	MakerEmail    string    `json:"maker_email"`
	CreatedAt     time.Time `json:"created_at"`
}

type IntegrationApp struct {
	IntegrationID string
	AppName       string
}

func (p *JSONPrices) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONPrices: value is not []byte")
	}

	return json.Unmarshal(bytes, p)
}

func (p JSONPrices) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *JSONSystemPrompts) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONSystem: value is not []byte")
	}

	return json.Unmarshal(bytes, p)
}

func (p JSONSystemPrompts) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *Provider) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("Scan failed: expected []byte but got %T", value)
	}
	return json.Unmarshal(bytes, p)
}

func (p Provider) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (s *JSONSkills) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("Scan failed: expected []byte but got %T", value)
	}
	return json.Unmarshal(bytes, s)
}

func (s JSONSkills) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s JSONSkills) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]Skill(s))
}

func (p *CapabilitiesObject) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONPrices: value is not []byte")
	}

	return json.Unmarshal(bytes, p)
}

func (p CapabilitiesObject) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func GenerateAgentKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (i *Integrations) CreateIntegration(db *gorm.DB, req Integrations) error {

	err := postgresql.CreateOneRecord(db, &i)
	if err != nil {
		return err
	}

	return nil
}

func (oi *OrganisationIntegrations) CreateOrganisationIntegration(db *gorm.DB) error {

	err := postgresql.CreateOneRecord(db, &oi)
	if err != nil {
		return err
	}

	return nil
}

func (i *Integrations) GetAllAgentApp(db *gorm.DB, org_id string, c *gin.Context) (AgentsResp, error) {

	var (
		agents AgentsResp
		org    Organisation
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", org_id)
	if !exists {
		return nil, errors.New("organisation not found")
	}

	err := db.Table("integrations AS i").
		Select(`i.id, i.name, i.app_logo, i.app_url, i.json_url, i.app_description, i.integration_type,
				i.is_system, 
				COALESCE(oi.created_at, i.created_at) AS created_at, 
				COALESCE(oi.updated_at, i.updated_at) AS updated_at, 
				COALESCE(oi.is_active, false) AS is_active, 
				CASE 
					WHEN oi.is_active IS TRUE THEN 'active' 
					ELSE 'inactive' 
				END AS status,
				CASE 
					WHEN oi.integration_id IS NOT NULL THEN true
					ELSE false 
				END AS linked`).
		Joins("LEFT JOIN organisation_integrations AS oi ON oi.integration_id = i.id AND oi.org_id = ?", org_id).
		Find(&agents).Error
	if err != nil {
		return nil, err
	}

	return agents, nil
}

// Get custom integrations
func (i *OrganisationIntegrations) GetCustomAgentApps(db *gorm.DB, org_id string, c *gin.Context) ([]OrganisationIntegrations, postgresql.PaginationResponse, error, int) {

	var (
		org        Organisation
		orgIntResp []OrganisationIntegrations
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", org_id)
	if !exists {
		return nil, postgresql.PaginationResponse{}, errors.New("organisation not found"), http.StatusNotFound
	}

	pagination := postgresql.GetPagination(c)

	query := db.Model(&OrganisationIntegrations{}).
		Where("org_id = ?", org_id)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&orgIntResp,
		nil,
	)
	if err != nil {
		return orgIntResp, paginationResponse, err, http.StatusInternalServerError
	}

	return orgIntResp, paginationResponse, err, http.StatusOK
}

func (i *Integrations) GetSystemAgentApps(db *gorm.DB, c *gin.Context) ([]Integrations, postgresql.PaginationResponse, error, int) {

	var (
		IntResp []Integrations
	)

	pagination := postgresql.GetPagination(c)

	query := db.Model(&Integrations{})

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&IntResp,
		nil,
	)
	if err != nil {
		return IntResp, paginationResponse, err, http.StatusInternalServerError
	}

	return IntResp, paginationResponse, err, http.StatusOK
}

func (i *Integrations) GetSystemAgentApp(db *gorm.DB, int_id string, c *gin.Context) (Integrations, error, int) {

	var (
		IntResp Integrations
	)

	exists := postgresql.CheckExists(db, &IntResp, "id = ?", int_id)
	if !exists {
		return IntResp, errors.New("Agent does not exist"), http.StatusNotFound
	}

	return IntResp, nil, http.StatusOK
}

func (i *Integrations) UpdateAgent(db *gorm.DB, ids map[string]string, req UpdateAgent) (Integrations, error) {
	var agent Integrations

	exists := postgresql.CheckExists(db, &agent, "id = ?", ids["agent_id"])
	if !exists {
		return agent, errors.New("agent app does not exist")
	}

	result, err := postgresql.UpdateFields(db, &agent, req, "id = ?", agent.ID)
	if err != nil {
		return agent, errors.New("failed to update agent app")
	}
	if result.RowsAffected == 0 {
		return agent, errors.New("no record updated")
	}

	updatedAgent := Integrations{}
	err = db.Where("id = ?", agent.ID).First(&updatedAgent).Error
	if err != nil {
		return updatedAgent, err
	}
	return updatedAgent, nil
}

// Delete general integration
func (i *Integrations) DeleteAgent(db *gorm.DB, ids map[string]string) error {
	var agent Integrations

	exists := postgresql.CheckExists(db, &agent, "id = ?", ids["agent_id"])
	if !exists {
		return errors.New("agent app does not exist")
	}

	err := db.Delete(&agent, "id = ?", ids["agent_id"]).Error
	if err != nil {
		return err
	}

	//also delete entries for the agent in the organisation agents table
	err = db.Delete(&OrganisationIntegrations{}, "integration_id = ?", ids["agent_id"]).Error
	if err != nil {
		return err
	}

	return nil
}

// Delete Custom integration
func (i *OrganisationIntegrations) DeleteCustomAgent(db *gorm.DB, logger utility.Logger, ids IDS) (error, int) {
	var (
		org_integration OrganisationIntegrations
		dmchannels      []DmChannels
		channelIDs      []string
		thread          Threads
	)

	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error), http.StatusInternalServerError
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	exists := postgresql.CheckExists(tx, &org_integration, "integration_id = ?", ids.AgentID)
	if !exists {
		tx.Rollback()
		return errors.New("agent app does not exist"), http.StatusBadRequest
	}

	//also delete entries for the agent in the organisation agents table
	err := tx.Delete(&OrganisationIntegrations{}, "integration_id = ?", ids.AgentID).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete organisation integration: %w", err), http.StatusInternalServerError
	}

	err = tx.Delete(&CustomIntegrationsSetting{}, "integration_id = ?", ids.AgentID).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete custom integration settings: %w", err), http.StatusInternalServerError
	}

	err = postgresql.SelectAllFromDb(tx, "", &dmchannels, "org_id = ? AND chat_type = 'bot' AND participant_id = ?", ids.OrganisationID, ids.AgentID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to fetch bot DM channels: %w", err), http.StatusInternalServerError
	}

	if len(dmchannels) > 0 {
		for _, channel := range dmchannels {
			channelIDs = append(channelIDs, channel.ChannelId)
		}

		err = postgresql.HardDeleteRecordFromDb(tx, &dmchannels)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete bot DM channels: %w", err), http.StatusInternalServerError
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err), http.StatusInternalServerError
	}

	//clear all threads related to the dm channel
	if len(channelIDs) > 0 {
		for _, channelID := range channelIDs {
			thread.ID = channelID
			_, err := thread.ClearThreadsByChannelID(db)
			if err != nil {
				logger.Error("Warning: Failed to clear threads for channel %s: %v", channelID, err)
			}
		}
	}

	return nil, http.StatusOK
}

func (oi *OrganisationIntegrations) UpdateJSONSchema(db *gorm.DB, req UpdateJSONSchemaRequest, ids map[string]string) error {

	update := make(map[string]any)
	update["json_schema"] = req.JSONSchema

	result, err := postgresql.UpdateFields(db, &oi, update, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("no record updated")
	}

	return nil
}

func (oi *OrganisationIntegrations) UpdateCustomAgent(db *gorm.DB, req CreateAgentRequest) (int, error) {

	update := make(map[string]any)
	update["title"] = req.Title
	update["app_name"] = req.Name
	update["app_description"] = req.Description
	update["visibility"] = req.Visibility
	update["app_logo"] = req.Avatar
	update["tone"] = req.Tone
	update["system_prompts"] = req.SystemPrompts

	result, err := postgresql.UpdateFields(db, &oi, update, "integration_id = ?", req.AgentId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if result.RowsAffected == 0 {
		return http.StatusOK, errors.New("no record updated")
	}

	return http.StatusOK, nil
}

func (oi *OrganisationIntegrations) UpdateCustomAgentPrompt(db *gorm.DB, req UpdateAgentPromptRequest) (int, error) {

	update := make(map[string]any)
	update["system_prompts"] = req.SystemPrompts

	result, err := postgresql.UpdateFields(db, &oi, update, "integration_id = ? AND org_id = ?", req.AgentId, req.OrgId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if result.RowsAffected == 0 {
		return http.StatusOK, errors.New("no record updated")
	}

	return http.StatusOK, nil
}

func (oi *OrganisationIntegrations) ChangeStatus(db *gorm.DB, req ChangeAgentStatus, ids map[string]string, extReq request.ExternalRequest) error {
	var (
		agent        Integrations
		organisation Organisation
		oci          OrganisationChannelsIntegrations
		channels     []Channels
		orgchannels  []OrganisationChannelsIntegrations
		intsettings  CustomIntegrationsSetting
	)

	organisationExists := postgresql.CheckExists(db, &organisation, "id = ?", ids["org_id"])
	if !organisationExists {
		return errors.New("organisation does not exist")
	}

	orgAgentExists := postgresql.CheckExists(db, &oi, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	agentExists := postgresql.CheckExists(db, &agent, "id = ?", ids["agent_id"])
	ChannelagentExists := postgresql.CheckExists(db, &oci, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	CheckIntegrationSettings := postgresql.CheckExists(db, &intsettings, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])

	if !(agentExists || orgAgentExists) {
		return errors.New("integration app does not exist")
	}

	//if the integration exists but does not have an entry in the organisation integrations table, create one
	if !orgAgentExists {
		oi.ID = utility.GenerateUUID()
		oi.IsActive = req.Status
		oi.OrgID = ids["org_id"]
		oi.IntegrationID = ids["agent_id"]
		oi.JSONSchema = req.JSONSchema
		oi.JSONUrl = agent.JSONUrl
		oi.AppDescription = agent.AppDescription
		oi.AppName = agent.Name
		oi.AppUrl = agent.AppUrl
		oi.AppLogo = agent.AppLogo
		oi.Prices = agent.Prices
		oi.Provider = agent.Provider
		oi.Version = agent.Version
		oi.DefaultInputModes = agent.DefaultInputModes
		oi.DefaultOutputModes = agent.DefaultOutputModes
		oi.Skills = agent.Skills
		oi.IsPaid = agent.IsPaid
		oi.IsSystem = true
		oi.PreSharedKey = agent.PreSharedKey
		oi.OwnerID = ids["user_id"]
		oi.SystemPrompts = agent.SystemPrompts
		oi.Title = agent.Title
		oi.Tone = agent.Tone

		err := oi.CreateOrganisationIntegration(db)
		if err != nil {
			return err
		}
	}

	//activate integration for all channels in the organisation
	if !ChannelagentExists {
		err := postgresql.SelectAllFromDb(db, "", &channels, "organisation_id = ?", ids["org_id"])
		if err != nil {
			return err
		}

		is_system := false

		if agentExists {
			is_system = true
		}

		for _, channel := range channels {
			oci := OrganisationChannelsIntegrations{
				ID:            utility.GenerateUUID(),
				OrgID:         ids["org_id"],
				ChannelID:     channel.ID,
				IntegrationID: ids["agent_id"],
				IsActive:      req.Status,
				IsSystem:      is_system,
			}

			orgchannels = append(orgchannels, oci)
		}

		err = postgresql.CreateMultipleRecords(db, &orgchannels, len(orgchannels))
		if err != nil {
			return err
		}
	}

	// add settings if not exist
	if !CheckIntegrationSettings {
		var agentSettings CustomIntegrationsSetting

		settings_data := map[string]any{"settings": ""}

		auth_credentials := map[string]any{"agent_auth_credentials": "Not-Set-Yet"}
		auth_credentials["agent_api_key"] = agent.PreSharedKey
		settings_data["auth_credentials"] = auth_credentials

		settingJsonData, err := json.Marshal(settings_data)
		if err != nil {
			return fmt.Errorf("error serializing to JSON: %v", err)
		}

		serialized_settings := string(settingJsonData)

		agentSettings.ID = utility.GenerateUUID()
		agentSettings.SettingEntry = serialized_settings
		agentSettings.OrgID = ids["org_id"]
		agentSettings.IsSystem = false
		agentSettings.IntegrationID = ids["agent_id"]

		err = agentSettings.CreateIntegrationSettings(db)
		if err != nil {
			return errors.New("failed to create agent settings")
		}
	}

	// Add the missing channels in a bulk insert without using a for loop @cyberguru
	err := db.Exec(`
			INSERT INTO organisation_channels_integrations (id, org_id, integration_id, channel_id, is_active, created_at, updated_at)
			SELECT gen_random_uuid(), ?, ?, c.id, ?, NOW(), NOW()
			FROM channels c
			WHERE c.organisation_id = ? 
			AND NOT EXISTS (
				SELECT 1 FROM organisation_channels_integrations oci 
				WHERE oci.channel_id = c.id AND oci.org_id = ? AND oci.integration_id = ?
		)`, ids["org_id"], ids["agent_id"], req.Status, ids["org_id"], ids["org_id"], ids["agent_id"]).Error

	if err != nil {
		return err
	}

	//when the integration has been deactivated/activated for the integration, deactivate/activate it for all channels in the organisation
	if req.Status || !req.Status {
		err := db.Model(&OrganisationChannelsIntegrations{}).
			Where("org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"]).
			Update("is_active", req.Status).Error
		if err != nil {
			return err
		}
	}

	update := make(map[string]any)
	update["is_active"] = req.Status
	update["json_schema"] = req.JSONSchema

	result, err := postgresql.UpdateFields(db, &oi, update, "org_id = ? AND integration_id = ?", oi.OrgID, oi.IntegrationID)
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("no record updated")
	}

	return nil
}

func (oci *OrganisationChannelsIntegrations) ChangeSendBackStatus(db *gorm.DB, req ChangeAgentStatus, ids map[string]string) error {
	var (
		integration  Integrations
		organisation Organisation
		channel      Channels
		oi           OrganisationIntegrations
	)

	organisationExists := postgresql.CheckExists(db, &organisation, "id = ?", ids["org_id"])
	if !organisationExists {
		return errors.New("organisation does not exist")
	}

	channelExists := postgresql.CheckExists(db, &channel, "id = ?", ids["channel_id"])
	if !channelExists {
		return errors.New("channel does not exist")
	}

	integrationExists := postgresql.CheckExists(db, &integration, "id = ?", ids["agent_id"])
	if !integrationExists {
		return errors.New("agent app does not exist")
	}

	orgAgentExists := postgresql.CheckExists(db, &oi, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])

	if !orgAgentExists {
		return errors.New("organisation not integrated with this app")
	}

	orgChannelAgentExists := postgresql.CheckExists(db, &oci, "is_active = ? AND org_id = ? AND integration_id = ? AND channel_id = ?", "true", ids["org_id"], ids["agent_id"], ids["channel_id"])

	if !orgChannelAgentExists {
		return errors.New("organisation not found or not active")
	}

	update := make(map[string]any)
	update["send_back"] = req.Status

	result, err := postgresql.UpdateFields(db, &oci, update, "org_id = ? AND integration_id = ? AND channel_id = ?", oci.OrgID, oci.IntegrationID, oci.ChannelID)

	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("agent not found or not active")
	}

	return nil
}

func (oci *OrganisationChannelsIntegrations) GetOrganisationChannelAgents(db *gorm.DB, channel_id, orgID string, c *gin.Context) ([]OrganisationIntegrations, postgresql.PaginationResponse, int, error) {
	var (
		org        Organisation
		orgIntResp []OrganisationIntegrations
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", orgID)
	if !exists {
		return nil, postgresql.PaginationResponse{}, http.StatusNotFound, errors.New("organisation not found")
	}

	pagination := postgresql.GetPagination(c)

	query := db.Table("organisation_channels_integrations AS c").
		Joins("JOIN organisation_integrations AS i ON c.integration_id = i.integration_id AND c.org_id = i.org_id").
		Where("c.org_id = ? AND c.channel_id = ? AND i.json_url != ''", orgID, channel_id).
		Select("c.id, c.org_id, c.integration_id, c.is_active, c.is_system, c.archived_at, " +
			"c.created_at, c.updated_at, i.json_url, i.app_name, i.app_url, i.app_logo, i.app_description")

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"c.created_at",
		"desc",
		pagination,
		&orgIntResp,
		nil,
	)

	if err != nil {
		return orgIntResp, paginationResponse, http.StatusInternalServerError, err
	}

	return orgIntResp, paginationResponse, http.StatusOK, err
}

func (oci *OrganisationChannelsIntegrations) ActivateChannelAgent(db *gorm.DB, req ActivateChannelAgent, ids map[string]string) error {

	exists := postgresql.CheckExists(db, &oci, "channel_id = ? AND org_id = ? AND integration_id = ?", ids["channel_id"], ids["organisation_id"], ids["agent_id"])

	if exists {

		update := make(map[string]any)
		update["is_active"] = req.Status

		result, err := postgresql.UpdateFields(db, &oci, update, "channel_id = ? AND org_id = ? AND integration_id = ?", ids["channel_id"], ids["organisation_id"], ids["agent_id"])

		if err != nil {
			return err
		}

		if result.RowsAffected == 0 {
			return errors.New("no record updated")
		}

	} else {

		ociInt := OrganisationChannelsIntegrations{
			ID:            utility.GenerateUUID(),
			OrgID:         ids["organisation_id"],
			ChannelID:     ids["channel_id"],
			IntegrationID: ids["agent_id"],
			IsActive:      req.Status,
		}

		err := ociInt.CreateOrganisationChannelIntegration(db)
		if err != nil {
			return err
		}

		return nil
	}
	return nil
}

func (oci *OrganisationChannelsIntegrations) CreateOrganisationChannelIntegration(db *gorm.DB) error {
	exists := postgresql.CheckExists(db, &oci, "channel_id = ? AND org_id = ? AND integration_id = ?", oci.ChannelID, oci.OrgID, oci.IntegrationID)

	if exists {
		return errors.New("organisation channel integration already exists")
	}

	err := postgresql.CreateOneRecord(db, &oci)
	if err != nil {
		return err
	}

	return nil
}

func (i *Integrations) CreateSlackIntegration(db *gorm.DB, name string) error {
	var integrationApp Integrations

	exists := postgresql.CheckExists(db, &integrationApp, "name = ?", name)

	if exists {
		return errors.New("slack integration already exists")
	}

	err := postgresql.CreateOneRecord(db, &i)
	if err != nil {
		return err
	}

	return nil
}

func (i *Integrations) GetIntegrationID(db *gorm.DB, name string) error {
	exists := postgresql.CheckExists(db, &i, "name = ?", name)

	if !exists {
		return errors.New("integration does not exists")
	}

	return nil
}

func (i *Integrations) PerformQueries(db *gorm.DB, channel_id string) ([]Integrations, error) {
	var (
		channelID = channel_id
		results   []Integrations
	)

	// Fetch active integrations with their associated settings for the given channel
	err := db.Table("organisation_channels_integrations").
		Joins("JOIN integrations ON organisation_channels_integrations.integration_id = integrations.id").
		Where("organisation_channels_integrations.channel_id = ? AND organisation_channels_integrations.is_active = ?", channelID, true).
		Select("integrations.id AS integration_id, integrations.*").
		Scan(&results).Error

	if err != nil {
		return []Integrations{}, err
	}

	return results, nil
}

func (oci *OrganisationChannelsIntegrations) CheckHasIntegrations(db *gorm.DB, channelID string) (bool, error) {

	exists := postgresql.CheckExists(db, &oci, "channel_id = ?", channelID)
	if !exists {
		return false, errors.New("channel integrations not found")
	}

	return true, nil
}

func (oci *OrganisationChannelsIntegrations) FetchIntegrationChannels(db *gorm.DB, ids map[string]string) (IntegrationChansResp, bool, error) {

	var res IntegrationChansResp

	orgId, intId := ids["organisation_id"], ids["agent_id"]

	err := db.Table("organisation_channels_integrations AS oci").
		Joins("JOIN channels ON channels.id = oci.channel_id").
		Where("oci.org_id = ? AND oci.integration_id = ?", orgId, intId).
		Select("oci.channel_id AS channel_id, channels.name AS channel_name, oci.is_active AS is_active").
		Scan(&res).Error

	exists := postgresql.CheckExists(db, &oci, "org_id = ? AND integration_id = ? AND is_active = FALSE", orgId, intId)

	if err != nil {
		return res, false, err
	}

	return res, exists, nil
}

func (i *OrganisationChannelsIntegrations) CheckIntegrationIsActive(db *gorm.DB, ids map[string]string) (bool, error) {

	var (
		organisation Organisation
		orgInt       OrganisationIntegrations
	)
	orgId, intId := ids["organisation_id"], ids["agent_id"]

	organisationExists := postgresql.CheckExists(db, &organisation, "id = ?", orgId)
	if !organisationExists {
		return false, errors.New("organisation does not exist")
	}

	exists := postgresql.CheckExists(db, &orgInt, "org_id = ? AND integration_id = ?", orgId, intId)
	if !exists {
		return false, errors.New("integration app does not exist")
	}

	return orgInt.IsActive, nil
}

func (oci *OrganisationChannelsIntegrations) CheckHasFilterIntegrations(db *gorm.DB, channelID string) (bool, error) {

	var count int64

	err := db.Table("organisation_channels_integrations AS oci").
		Joins("JOIN organisation_integrations AS oi ON oi.org_id = oci.org_id AND oi.is_active = 't' AND oi.integration_id = oci.integration_id ").
		Joins("LEFT JOIN integrations ON oci.integration_id = integrations.id").
		Where("oci.channel_id = ? AND oci.is_active = ?", channelID, true).
		Select("integrations.id AS integration_id, integrations.*").
		Count(&count).Error

	if err != nil {
		return false, err
	}

	if count == 0 {
		return false, nil
	}

	return true, nil
}

func (ic *IntegrationChannel) CreateIntegrationChan(db *gorm.DB, int_out_id string) (IntegrationChannel, int, error) {

	var (
		intchan IntegrationChannel
		intMod  Integrations
		int_out IntegrationOutput
	)

	exist := postgresql.CheckExists(db, &intchan, "int_channel_id = ? AND integration_id = ? AND channel_id = ? AND output_id = ?", ic.IntChannelID, ic.IntegrationID, ic.ChannelID, int_out_id)

	if exist {
		return intchan, http.StatusCreated, nil
	}

	exist = postgresql.CheckExists(db, &intMod, "id = ? AND integration_type = ?", ic.IntegrationID, "m")

	if !exist {
		return intchan, http.StatusNotFound, fmt.Errorf("invalid integration id or modifier type, integration does not exist")
	}

	exists := postgresql.CheckExists(db, &int_out, "integration_modifier_id = ? AND channel_id = ? AND integration_output_id = ?", ic.IntegrationID, ic.ChannelID, int_out_id)
	if !exists {

		var (
			intOut Integrations
		)

		exist := postgresql.CheckExists(db, &intOut, "id = ? AND integration_type = ?", int_out_id, "o")

		if !exist {
			return intchan, http.StatusNotFound, fmt.Errorf("invalid integration id or output type, integration does not exist")
		}

		int_out = IntegrationOutput{
			ID:                    utility.GenerateUUID(),
			IntegrationOutputID:   int_out_id,
			IntegrationModifierID: ic.IntegrationID,
			IntegrationName:       intOut.Name,
			ChannelID:             ic.ChannelID,
			SendBack:              true,
		}

		err := postgresql.CreateOneRecord(db, &int_out)
		if err != nil {
			return intchan, http.StatusInternalServerError, err
		}
	}

	ic.IntegrationOutputID = int_out.ID
	err := postgresql.CreateOneRecord(db, &ic)
	if err != nil {
		return *ic, http.StatusInternalServerError, err
	}

	return *ic, http.StatusCreated, nil
}

func (ic *IntegrationChannel) GetIntegrationChannels(db *gorm.DB) ([]IntegrationOutput, int, error) {
	var (
		res    []IntegrationOutput
		intMod Integrations
	)

	exist := postgresql.CheckExists(db, &intMod, "id = ? AND integration_type = ?", ic.IntegrationID, "m")

	if !exist {
		return res, http.StatusNotFound, fmt.Errorf("invalid integration id or modifier type, integration does not exist")
	}

	err := db.Preload("IntegrationChannels").
		Where("integration_outputs.channel_id = ? AND integration_outputs.integration_modifier_id = ?", ic.ChannelID, ic.IntegrationID).
		Find(&res).Error

	if err != nil {
		return res, http.StatusInternalServerError, err
	}

	return res, http.StatusOK, err
}

func (ic *IntegrationChannel) DeleteChannelIntegration(db *gorm.DB, req IntegrationChannelReq) (int, error) {

	var intchan IntegrationChannel

	exist := postgresql.CheckExists(db, &intchan, "int_channel_id = ? AND integration_id = ? AND channel_id = ? AND output_id = ?", req.IntChannelID, req.IntegrationModifierID, req.ChannelID, req.IntegrationOutputID)

	if !exist {
		return http.StatusNotFound, fmt.Errorf("entry does not exist")
	}

	err := postgresql.DeleteRecordFromDb(db, intchan)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	var intcheck IntegrationChannel

	exist = postgresql.CheckExists(db, &intcheck, "integration_id = ? AND channel_id = ? AND output_id = ?", req.IntegrationModifierID, req.ChannelID, req.IntegrationOutputID)

	if !exist {

		var int_out IntegrationOutput

		exists := postgresql.CheckExists(db, &int_out, "integration_modifier_id = ? AND channel_id = ? AND integration_output_id = ?", req.IntegrationModifierID, req.ChannelID, req.IntegrationOutputID)

		if exists {

			err := postgresql.DeleteRecordFromDb(db, int_out)

			if err != nil {
				return http.StatusInternalServerError, err
			}
		}
	}

	return http.StatusOK, nil
}

// Custom Integration Settings CRUD

func (i *CustomIntegrationsSetting) CreateIntegrationSettings(db *gorm.DB) error {

	err := postgresql.CreateOneRecord(db, &i)
	if err != nil {
		return err
	}

	return nil
}

func (oi *CustomIntegrationsSetting) UpdateCustomIntegrationSettings(db *gorm.DB, req CustomIntegrationSettingRequest, ids map[string]string) error {

	update := make(map[string]any)

	deserialize_settings := make(map[string]any)

	var ucis CustomIntegrationsSetting

	// fetch existing settings
	exists := postgresql.CheckExists(db, &ucis, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	if !exists {
		return errors.New("integration not connnected yet")
	}

	settings := ucis.SettingEntry

	// unserialize the settings text
	err := json.Unmarshal([]byte(settings), &deserialize_settings)

	// update the important field (settings)

	deserialize_settings["settings"] = req.SettingEntry["settings"]

	auth_creds, ok := req.SettingEntry["auth_credentials"]

	if ok {
		deserialize_settings["auth_credentials"] = auth_creds

		encoded_auth_cred, ok := auth_creds.(map[string]any)["integration_auth_credentials"].(string)

		if ok {

			_, err := base64.StdEncoding.DecodeString(encoded_auth_cred)
			if err != nil {
				return fmt.Errorf("invalid integration_auth_credentials supplied, ensure it's base64 encoded")
			}

		} else {
			return fmt.Errorf("intergration_auth_credentials field is missing")
		}
	}

	settingJsonData, err := json.Marshal(deserialize_settings)

	if err != nil {
		return fmt.Errorf("error serializing to JSON: %v", err)
	}

	serialized_settings := string(settingJsonData)

	update["setting_entry"] = serialized_settings

	result, err := postgresql.UpdateFields(db, &oi, update, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("no record updated")
	}

	return nil
}

func ValidateAgentData(data_r map[string]any) error {

	var categories = map[string]bool{
		"Monitoring & Logging":           true,
		"Communication & Collaboration":  true,
		"Security & Compliance":          true,
		"Performance Monitoring":         true,
		"Website Uptime":                 true,
		"Social Media Management":        true,
		"CRM & Customer Support":         true,
		"Marketing Automation":           true,
		"Data Analytics & Visualization": true,
		"Finance & Payments":             true,
		"Project Management":             true,
		"E-commerce & Retail":            true,
		"AI & Machine Learning":          true,
		"Task Automation":                true,
		"Cloud Services":                 true,
		"Human Resources & Payroll":      true,
		"Email & Messaging":              true,
		"IT Service Management":          true,
		"Development & Code Management":  true,
		"DevOps & CI/CD":                 true,
	}

	_ = categories

	app_name, ok := data_r["name"].(string)
	if !ok || app_name == "" {
		return errors.New("Failed to save agent, invalid agent card: name field does not exist.")
	}

	// app_logo, ok := descriptions["app_logo"].(string)
	// if !ok || app_logo == "" {
	// 	return errors.New("Failed to save agent, app_logo field does not exist or is empty")
	// }

	// if !strings.Contains(app_logo, "https:") && !strings.Contains(app_logo, "http:") {
	// 	return errors.New("Failed to save agent, invalid app_logo url")
	// }

	app_url, ok := data_r["url"].(string)
	if !ok || app_url == "" {
		return errors.New("Failed to save agent, invalid agent card: url field does not exist or is empty")
	}

	skills, ok := data_r["skills"]
	if !ok {
		return errors.New("Failed to save agent, skills field does not exist or is empty")
	}

	skillsBytes, err := json.Marshal(skills)
	if err != nil {
		return err
	}

	var skillsObj []Skill
	if err := json.Unmarshal(skillsBytes, &skillsObj); err != nil {
		return errors.New("failed to save agent: 'skills' field is invalid")
	}

	defaultInputModes, ok := data_r["defaultInputModes"]
	if !ok {
		return errors.New("Failed to save agent, defaultInputModes field does not exist or is empty")
	}

	_, ok = defaultInputModes.([]any)
	if !ok {
		return errors.New("Failed to save agent, defaultInputModes field is not an array")
	}

	defaultOutputModes, ok := data_r["defaultOutputModes"]
	if !ok {
		return errors.New("Failed to save agent, defaultOutputModes field does not exist or is empty")
	}
	_, ok = defaultOutputModes.([]any)
	if !ok {
		return errors.New("Failed to save agent, defaultOutputModes field is not an array")
	}

	providerMap, ok := data_r["provider"].(map[string]any)
	if !ok {
		return errors.New("Failed to save agent, invalid agent card: provider does not exist or is empty")
	}

	providerBytes, err := json.Marshal(providerMap)
	if err != nil {
		return err
	}

	var provider Provider
	if err := json.Unmarshal(providerBytes, &provider); err != nil {
		return err
	}

	return nil
}

func (cis *CustomIntegrationsSetting) FetchAPIKey(db *gorm.DB, ids IDS) (string, int, error) {

	var (
		agent OrganisationIntegrations
		org   Organisation
	)

	exist := postgresql.CheckExists(db, &org, "id = ?", ids.OrganisationID)
	if !exist {
		return "", http.StatusNotFound, errors.New("organisation not found")
	}

	if org.OwnerID != ids.UserID {
		return "", http.StatusForbidden, errors.New("user not allowed to fetch agent's settings")
	}

	exists := postgresql.CheckExists(db, &agent, "integration_id = ?", ids.AgentID)
	if !exists {
		return "", http.StatusNotFound, errors.New("agent app does not exist")
	}

	err := db.Model(&cis).Where("org_id = ? AND integration_id = ?", ids.OrganisationID, ids.AgentID).Select("setting_entry").First(&cis).Error
	if err != nil {
		return "", http.StatusInternalServerError, errors.New("failed to fetch agent settings")
	}

	return cis.SettingEntry, http.StatusOK, nil
}

func ValidateAgentVersionAndUpdate(data_r map[string]any, agentID, db *gorm.DB) error {
	var agent OrganisationIntegrations

	version, ok := data_r["version"].(string)
	if !ok {
		return errors.New("failed to save agent: 'version' field is missing or invalid")
	}

	err := db.Where("id = ?", agentID).First(&agent).Error
	if err != nil {
		return fmt.Errorf("agent not found: %v", err)
	}

	if agent.Version != version {
		if appName, ok := data_r["app_name"].(string); ok {
			agent.AppName = appName
		}

		if appDesc, ok := data_r["app_description"].(string); ok {
			agent.AppDescription = appDesc
		}

		if appLogo, ok := data_r["app_logo"].(string); ok {
			agent.AppLogo = appLogo
		}

		if provider, ok := data_r["provider"].(Provider); ok {
			agent.Provider = provider
		}

		isPaid, _ := data_r["is_paid"].(bool)

		if isPaid {
			rawPrices, ok := data_r["prices"]
			if !ok {
				return errors.New("failed to save agent: 'prices' field is missing")
			}

			jsonBytes, err := json.Marshal(rawPrices)
			if err != nil {
				return errors.New("failed to save agent: 'prices' field could not be marshaled")
			}

			var prices JSONPrices
			if err := json.Unmarshal(jsonBytes, &prices); err != nil {
				return errors.New("failed to save agent: 'prices' field is invalid")
			}

			agent.Prices = prices
		}

		agent.Version = version

		// Save updates
		if err := db.Save(&agent).Error; err != nil {
			return fmt.Errorf("failed to update agent info: %v", err)
		}
	}

	return nil
}

func UpdateCustomAgent(db *gorm.DB, ids map[string]string) error {
	var (
		agentSettings CustomIntegrationsSetting
		agent         OrganisationIntegrations
		extReq        request.ExternalRequest
	)

	if err := db.Where("org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"]).First(&agent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("agent with org_id %s and integration_id %s does not exist", ids["org_id"], ids["agent_id"])
		}
		return err
	}

	data := map[string]string{"url": agent.JSONUrl}
	response, _ := extReq.SendExternalRequest(request.AgentJsonContent, data)

	data_r, _ := response.(map[string]any)

	// Only generate new pre-shared key if agent does not it yet
	if agent.PreSharedKey == "" {
		psk, err := GenerateAgentKey()
		if err != nil {
			return err
		}
		agent.PreSharedKey = psk
	}

	bytes, err := json.Marshal(data_r)
	if err != nil {
		return err
	}

	var payload OrganisationIntegrations
	json.Unmarshal(bytes, &payload)

	agent.AppName = data_r["name"].(string)
	agent.AppDescription = data_r["description"].(string)
	agent.AppUrl = data_r["url"].(string)
	agent.Prices = payload.Prices
	agent.Provider = payload.Provider
	agent.Version = payload.Version
	agent.DefaultInputModes = payload.DefaultInputModes
	agent.DefaultOutputModes = payload.DefaultOutputModes
	agent.Skills = payload.Skills
	agent.IsPaid = payload.IsPaid

	if err := db.Save(&agent).Error; err != nil {
		return err
	}

	// Find and update existing custom agent settings
	err = db.Where("org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"]).First(&agentSettings).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("agent settings for org_id %s and integration_id %s not found", ids["org_id"], ids["agent_id"])
		}
		return fmt.Errorf("error querying agent settings: %v", err)
	}

	// Update SettingEntry field with new serialized {agent api key}
	settingsData := map[string]any{"settings": ""}
	authCredentials := map[string]any{
		"agent_auth_credentials": "Not-Set-Yet",
		"agent_api_key":          agent.PreSharedKey,
	}
	settingsData["auth_credentials"] = authCredentials

	settingJsonData, err := json.Marshal(settingsData)
	if err != nil {
		return fmt.Errorf("error serializing settings to JSON: %v", err)
	}

	agentSettings.SettingEntry = string(settingJsonData)

	if err := db.Save(&agentSettings).Error; err != nil {
		return fmt.Errorf("error updating agent settings: %v", err)
	}

	return nil
}

func ValidateAgentApiKey(db *gorm.DB, apiKey string) (string, string, error) {
	var agentSettings CustomIntegrationsSetting

	err := db.
		Where("setting_entry::jsonb -> 'auth_credentials' ->> 'agent_api_key' = ?", apiKey).
		First(&agentSettings).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", fmt.Errorf("no agent settings found for provided api key")
		}
		return "", "", fmt.Errorf("error querying agent settings: %v", err)
	}

	return agentSettings.OrgID, agentSettings.IntegrationID, nil
}

func GetAgentsByOwner(db *gorm.DB, user_id string) ([]OrganisationIntegrations, error) {
	var agents []OrganisationIntegrations

	err := db.Where("owner_id = ?", user_id).Find(&agents).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get integration settings: %v", err)
	}
	return agents, nil
}

func (i *PartialOrganisationIntegration) GetAllSystemAgent(
	db *gorm.DB,
	c *gin.Context,
	search string,
	sortBy string,
	sortOrder string,
	active bool,
) ([]PartialOrganisationIntegration, postgresql.PaginationResponse, error, int) {

	var results []Integrations

	pagination := postgresql.GetPagination(c)

	query := db.Model(&Integrations{}).
		Where("json_url != ''")

	if search != "" {
		searchValue := "%" + search + "%"
		query = query.Where("name ILIKE ?", searchValue)
	}

	if active {
		query = query.Where("is_active = ?", true)
	} else {
		query = query.Where("is_active = ?", false)
	}

	if sortBy == "" || sortBy == "credit_used" {
		sortBy = "created_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		sortBy,
		sortOrder,
		pagination,
		&results,
		nil,
	)

	IntResp := make([]PartialOrganisationIntegration, len(results))
	for i, int_mp := range results {
		IntResp[i] = PartialOrganisationIntegration{
			ID:         int_mp.ID,
			Name:       int_mp.Name,
			IsActive:   int_mp.IsActive,
			IsPaid:     int_mp.IsPaid,
			Provider:   int_mp.Provider,
			CreatedAt:  int_mp.CreatedAt,
			JSONUrl:    int_mp.JSONUrl,
			Source:     "non-organization",
			IsSystem:   int_mp.IsSystem,
			IsApproved: int_mp.IsApproved,
		}
	}

	if err != nil {
		return IntResp, paginationResponse, err, http.StatusInternalServerError
	}

	agentIDs := make([]string, len(IntResp))
	for i, agent := range IntResp {
		agentIDs[i] = agent.ID
	}

	var creditAggregates []CreditAggregate
	err = db.Model(&CreditUsage{}).
		Select("agent_id AS integration_id, COALESCE(SUM(amount), 0) AS total_used").
		Where("agent_id IN ?", agentIDs).
		Group("agent_id").
		Scan(&creditAggregates).Error

	if err != nil {
		return nil, paginationResponse, err, http.StatusInternalServerError
	}

	creditMap := map[string]float64{}
	for _, ca := range creditAggregates {
		creditMap[ca.IntegrationID] = ca.TotalUsed
	}

	for i := range IntResp {
		if total, ok := creditMap[IntResp[i].ID]; ok {
			IntResp[i].CreditUsed = total
		}
	}

	if sortBy == "credit_used" {
		sort.SliceStable(IntResp, func(i, j int) bool {
			if sortOrder == "asc" {
				return IntResp[i].CreditUsed < IntResp[j].CreditUsed
			}
			return IntResp[i].CreditUsed > IntResp[j].CreditUsed
		})
	}

	return IntResp, paginationResponse, nil, http.StatusOK
}

func (i *PartialOrganisationIntegration) GetAllCustomAgent(
	db *gorm.DB,
	c *gin.Context,
	search string,
	sortBy string,
	sortOrder string,
	active bool,
) ([]PartialOrganisationIntegration, postgresql.PaginationResponse, error, int) {

	var results []OrganisationIntegrations

	pagination := postgresql.GetPagination(c)

	query := db.Model(&OrganisationIntegrations{}).
		Where("json_url != ''")

	if search != "" {
		searchValue := "%" + search + "%"
		query = query.Where("app_name ILIKE ?", searchValue)
	}

	if active {
		query = query.Where("is_active = ?", true)
	} else {
		query = query.Where("is_active = ?", false)
	}

	if sortBy == "" || sortBy == "credit_used" {
		sortBy = "created_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		sortBy,
		sortOrder,
		pagination,
		&results,
		nil,
	)

	orgIntResp := make([]PartialOrganisationIntegration, len(results))
	for i, org := range results {
		orgIntResp[i] = PartialOrganisationIntegration{
			ID:            org.ID,
			AppName:       org.AppName,
			IntegrationID: &org.IntegrationID,
			IsActive:      org.IsActive,
			IsPaid:        org.IsPaid,
			Provider:      org.Provider,
			CreatedAt:     org.CreatedAt,
			JSONUrl:       org.JSONUrl,
			Source:        "organization",
			IsSystem:      org.IsSystem,
			IsApproved:    org.IsApproved,
			OrgID:         &org.OrgID,
		}
	}

	if err != nil {
		return orgIntResp, paginationResponse, err, http.StatusInternalServerError
	}

	agentIDs := make([]string, len(orgIntResp))
	for i, agent := range orgIntResp {
		agentIDs[i] = *agent.IntegrationID
	}

	var creditAggregates []CreditAggregate
	err = db.Model(&CreditUsage{}).
		Select("agent_id as integration_id,  COALESCE(SUM(amount), 0) AS total_used").
		Where("agent_id IN ?", agentIDs).
		Group("agent_id").
		Scan(&creditAggregates).Error

	if err != nil {
		return nil, paginationResponse, err, http.StatusInternalServerError
	}

	creditMap := map[string]float64{}
	for _, ca := range creditAggregates {
		creditMap[ca.IntegrationID] = ca.TotalUsed
	}

	for i := range orgIntResp {
		if total, ok := creditMap[*orgIntResp[i].IntegrationID]; ok {
			orgIntResp[i].CreditUsed = total
		}
	}

	if sortBy == "credit_used" {
		sort.SliceStable(orgIntResp, func(i, j int) bool {
			if sortOrder == "asc" {
				return orgIntResp[i].CreditUsed < orgIntResp[j].CreditUsed
			}
			return orgIntResp[i].CreditUsed > orgIntResp[j].CreditUsed
		})
	}

	// attach agent credit earned
	var agentBillAggregates []AgentBillAggregate
	err = db.Model(&IntegrationBills{}).
		Select("integration_id as integration_id,  COALESCE(SUM(maker_amount), 0) AS maker_total_earning, COALESCE(SUM(telex_amount), 0) AS telex_total_earning").
		Where("integration_id IN ?", agentIDs).
		Group("integration_id").
		Scan(&agentBillAggregates).Error

	if err != nil {
		return nil, paginationResponse, err, http.StatusInternalServerError
	}

	billMap := map[string]Earnings{}
	for _, ca := range agentBillAggregates {
		billMap[ca.IntegrationID] = Earnings{
			MakerTotal: ca.MakerTotalEarning,
			TelexTotal: ca.TelexTotalEarning,
		}
	}

	for i := range orgIntResp {
		id := *orgIntResp[i].IntegrationID
		if earnings, ok := billMap[id]; ok {
			orgIntResp[i].MakerTotalEarning = earnings.MakerTotal
			orgIntResp[i].TelexTotalEarning = earnings.TelexTotal
		}
	}

	return orgIntResp, paginationResponse, nil, http.StatusOK
}

func (i *OrganisationIntegrations) GetCustomAgentCountMetrics(db *gorm.DB) (CustomIntegrationsMetrics, error) {
	var metrics CustomIntegrationsMetrics

	organisations := db.Model(&Organisation{})
	credits := db.Model(&CreditUsage{})

	if err := db.Model(&OrganisationIntegrations{}).Count(&metrics.All).Error; err != nil {
		return metrics, err
	}

	if err := db.Model(&OrganisationIntegrations{}).Where("is_active = ?", true).Count(&metrics.Active).Error; err != nil {
		return metrics, err
	}

	if err := db.Model(&OrganisationIntegrations{}).Where("is_active = ?", false).Count(&metrics.Inactive).Error; err != nil {
		return metrics, err
	}

	if err := organisations.Count(&metrics.Organizations).Error; err != nil {
		return metrics, err
	}

	if err := credits.Select("SUM(amount)").Scan(&metrics.Credits).Error; err != nil {
		return metrics, err
	}

	return metrics, nil
}

func (i *OrganisationIntegrations) GetCustomAgentByID(db *gorm.DB, agentID string) (AdminCustomAgentResp, error) {
	var resp AdminCustomAgentResp

	if err := db.Where("integration_id = ?", agentID).First(&resp.Agent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AdminCustomAgentResp{}, errors.New("agent not found")
		}
		return AdminCustomAgentResp{}, err
	}

	if err := postgresql.SelectAllFromDb(db, "", &resp.User, "id = ?", resp.Agent.OwnerID); err != nil {
		return AdminCustomAgentResp{}, fmt.Errorf("failed to get agent owner: %v", err)
	}

	var total float64
	if err := db.Table("credit_usages").
		Select("COALESCE(SUM(amount), 0)").
		Where("agent_id = ?", agentID).Scan(&total).Error; err != nil {
		return AdminCustomAgentResp{}, fmt.Errorf("failed to get total credit usage: %v", err)
	}

	resp.CreditUsed = total

	var agentBills AgentBillAggregate
	err := db.Model(&IntegrationBills{}).
		Select("integration_id, COALESCE(SUM(maker_amount), 0) AS maker_total_earning, COALESCE(SUM(telex_amount), 0) AS telex_total_earning").
		Where("integration_id = ?", agentID).
		Group("integration_id").
		Take(&agentBills).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return AdminCustomAgentResp{}, err
	}

	resp.MakerTotalEarning = agentBills.MakerTotalEarning
	resp.TelexTotalEarning = agentBills.TelexTotalEarning

	return resp, nil
}

func (i *OrganisationIntegrations) AdminDeleteCustomAgentApp(db *gorm.DB, logger utility.Logger, agentID string) (error, int) {
	var (
		org_integration OrganisationIntegrations
		dmchannels      []DmChannels
		channelIDs      []string
		thread          Threads
	)

	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error), http.StatusInternalServerError
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	exists := postgresql.CheckExists(tx, &org_integration, "integration_id = ?", agentID)
	if !exists {
		tx.Rollback()
		return errors.New("agent app does not exist"), http.StatusBadRequest
	}

	err := tx.Delete(&OrganisationIntegrations{}, "integration_id = ?", agentID).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete organisation integration: %w", err), http.StatusInternalServerError
	}

	err = tx.Delete(&CustomIntegrationsSetting{}, "integration_id = ?", agentID).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete custom integration settings: %w", err), http.StatusInternalServerError
	}

	err = tx.Delete(&OrganisationChannelsIntegrations{}, "integration_id = ?", agentID).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete organisation channels integration: %w", err), http.StatusInternalServerError
	}

	err = postgresql.SelectAllFromDb(tx, "", &dmchannels, "org_id = ? AND chat_type = 'bot' AND participant_id = ?", org_integration.OrgID, agentID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to fetch bot DM channels: %w", err), http.StatusInternalServerError
	}

	if len(dmchannels) > 0 {
		for _, channel := range dmchannels {
			channelIDs = append(channelIDs, channel.ChannelId)
		}

		err = postgresql.HardDeleteRecordFromDb(tx, &dmchannels)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete bot DM channels: %w", err), http.StatusInternalServerError
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err), http.StatusInternalServerError
	}

	if len(channelIDs) > 0 {
		for _, channelID := range channelIDs {
			thread.ID = channelID
			_, err := thread.ClearThreadsByChannelID(db)
			if err != nil {
				logger.Error("Warning: Failed to clear threads for channel %s: %v", channelID, err)
			}
		}
	}

	return nil, http.StatusOK
}

func (i *OrganisationIntegrations) AdminUpdateAgent(db *gorm.DB, agentID string, req AdminUpdateAgent) (OrganisationIntegrations, error) {
	var agent OrganisationIntegrations

	exists := postgresql.CheckExists(db, &agent, "integration_id = ?", agentID)
	if !exists {
		return agent, errors.New("agent app does not exist")
	}

	agent.IsActive = req.IsActive
	agent.IsApproved = req.IsApproved
	agent.IsSystem = req.IsSystem

	if err := db.Save(&agent).Error; err != nil {
		return agent, err
	}

	return agent, nil
}

func (i *Integrations) GetAgentByID(db *gorm.DB, agentID string) (AdminAgentResp, error) {
	var resp AdminAgentResp

	if err := db.Where("id = ?", agentID).First(&resp.Agent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AdminAgentResp{}, errors.New("agent not found")
		}
		return AdminAgentResp{}, err
	}

	if err := postgresql.SelectAllFromDb(db, "", &resp.User, "id = ?", resp.Agent.OwnerID); err != nil {
		return AdminAgentResp{}, fmt.Errorf("failed to get agent owner: %v", err)
	}

	var total float64
	if err := db.Table("credit_usages").
		Select("COALESCE(SUM(amount), 0)").
		Where("agent_id = ?", agentID).Scan(&total).Error; err != nil {
		return AdminAgentResp{}, fmt.Errorf("failed to get total credit usage: %v", err)
	}

	resp.CreditUsed = total

	return resp, nil
}

func (si *Integrations) CreateSystemIntegration(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &si)
	if err != nil {
		return err
	}

	return nil
}

func (i *IntegrationSettings) CreateSystemIntegrationSettings(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &i)
	if err != nil {
		return err
	}

	return nil
}

func (i *OrganisationIntegrations) CreateOrUpdateBillFromUsage(db *gorm.DB, usage *CreditUsage) error {
	var agent OrganisationIntegrations
	if err := db.Where("integration_id = ?", usage.AgentID).First(&agent).Error; err != nil {
		return fmt.Errorf("failed to fetch agent: %w", err)
	}

	commissionRate := agent.CommissionRate
	creditAmount := usage.Amount

	makerAmount := creditAmount * (commissionRate / 100)
	telexAmount := creditAmount - makerAmount
	totalAmount := creditAmount

	var existingBill IntegrationBills

	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	err := db.Where("integration_id = ? AND org_id = ? AND created_at >= ? AND created_at < ?",
		usage.AgentID, usage.OrganisationID, startOfDay, endOfDay).
		First(&existingBill).Error

	if err == gorm.ErrRecordNotFound {
		newBill := IntegrationBills{
			ID:            utility.GenerateUUID(),
			OrgID:         usage.OrganisationID,
			IntegrationID: usage.AgentID,
			MakerID:       agent.OwnerID,
			CreditUsageID: usage.ID,
			TelexAmount:   telexAmount,
			MakerAmount:   makerAmount,
			TotalAmount:   totalAmount,
		}

		if err := db.Create(&newBill).Error; err != nil {
			return fmt.Errorf("failed to create new bill: %w", err)
		}
	} else if err == nil {
		existingBill.TelexAmount += telexAmount
		existingBill.MakerAmount += makerAmount
		existingBill.TotalAmount += totalAmount

		if err := db.Save(&existingBill).Error; err != nil {
			return fmt.Errorf("failed to update existing bill: %w", err)
		}
	} else {
		return fmt.Errorf("failed to query bill: %w", err)
	}

	return nil
}

func (i *IntegrationBillsResponse) GetAgentBills(
	db *gorm.DB,
	c *gin.Context,
) ([]IntegrationBillsResponse, postgresql.PaginationResponse, error, int) {
	agent_id := c.Query("agent_id")
	payout_status := c.Query("payout_status")

	var intBills []IntegrationBills
	var agentBillResponses []IntegrationBillsResponse

	pagination := postgresql.GetPagination(c)

	query := db.Model(&IntegrationBills{}).
		Preload("Organisation")

	if agent_id != "" {
		query = query.Where("integration_id = ?", agent_id)
	}

	if payout_status != "" {
		query = query.Where("payout_status = ?", payout_status)
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&intBills,
		nil,
	)

	if err != nil {
		return agentBillResponses, paginationResponse, err, http.StatusInternalServerError
	}

	for _, bill := range intBills {
		agentBillResponses = append(agentBillResponses, IntegrationBillsResponse{
			ID:            bill.ID,
			OrgID:         bill.OrgID,
			IntegrationID: bill.IntegrationID,
			CreditUsageID: bill.CreditUsageID,
			TelexAmount:   bill.TelexAmount,
			MakerAmount:   bill.MakerAmount,
			TotalAmount:   bill.TotalAmount,
			PayoutStatus:  bill.PayoutStatus,
			OrgName:       bill.Organisation.Name,
			OrgEmail:      bill.Organisation.Email,
			MakerName:     bill.User.Name,
			MakerEmail:    bill.User.Email,
			CreatedAt:     bill.CreatedAt,
		})
	}

	agentIDs := make([]string, len(agentBillResponses))
	for i, agent := range agentBillResponses {
		agentIDs[i] = *&agent.IntegrationID
	}

	var integrationApps []IntegrationApp

	err = db.Model(&OrganisationIntegrations{}).
		Select("integration_id, app_name").
		Where("integration_id IN ?", agentIDs).
		Scan(&integrationApps).Error

	if err != nil {
		return nil, paginationResponse, err, http.StatusInternalServerError
	}

	appMap := make(map[string]string)
	for _, app := range integrationApps {
		appMap[app.IntegrationID] = app.AppName
	}

	for i, response := range agentBillResponses {
		if appName, ok := appMap[response.IntegrationID]; ok {
			agentBillResponses[i].AppName = appName
		}
	}

	return agentBillResponses, paginationResponse, nil, http.StatusOK
}

func (i *IntegrationBillsResponse) GetOrgAgentBills(
	db *gorm.DB,
	c *gin.Context,
	org_id string,
) ([]IntegrationBillsResponse, postgresql.PaginationResponse, error, int) {
	agent_id := c.Query("agent_id")
	payout_status := c.Query("payout_status")

	var intBills []IntegrationBills
	var agentBillResponses []IntegrationBillsResponse

	pagination := postgresql.GetPagination(c)

	query := db.Model(&IntegrationBills{}).
		Where("org_id = ?", org_id).
		Preload("Organisation")

	if agent_id != "" {
		query = query.Where("integration_id = ?", agent_id)
	}

	if payout_status != "" {
		query = query.Where("payout_status = ?", payout_status)
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&intBills,
		nil,
	)

	if err != nil {
		return agentBillResponses, paginationResponse, err, http.StatusInternalServerError
	}

	for _, bill := range intBills {
		agentBillResponses = append(agentBillResponses, IntegrationBillsResponse{
			ID:            bill.ID,
			OrgID:         bill.OrgID,
			IntegrationID: bill.IntegrationID,
			CreditUsageID: bill.CreditUsageID,
			TelexAmount:   bill.TelexAmount,
			MakerAmount:   bill.MakerAmount,
			TotalAmount:   bill.TotalAmount,
			PayoutStatus:  bill.PayoutStatus,
			OrgName:       bill.Organisation.Name,
			OrgEmail:      bill.Organisation.Email,
			MakerName:     bill.User.Name,
			MakerEmail:    bill.User.Email,
			CreatedAt:     bill.CreatedAt,
		})
	}

	agentIDs := make([]string, len(agentBillResponses))
	for i, agent := range agentBillResponses {
		agentIDs[i] = *&agent.IntegrationID
	}

	var integrationApps []IntegrationApp

	err = db.Model(&OrganisationIntegrations{}).
		Select("integration_id, app_name").
		Where("integration_id IN ?", agentIDs).
		Scan(&integrationApps).Error

	if err != nil {
		return nil, paginationResponse, err, http.StatusInternalServerError
	}

	appMap := make(map[string]string)
	for _, app := range integrationApps {
		appMap[app.IntegrationID] = app.AppName
	}

	for i, response := range agentBillResponses {
		if appName, ok := appMap[response.IntegrationID]; ok {
			agentBillResponses[i].AppName = appName
		}
	}

	return agentBillResponses, paginationResponse, nil, http.StatusOK
}

func (i *Integrations) AdminDeleteSystemAgentApp(db *gorm.DB, logger utility.Logger, agentID string) (error, int) {
	var (
		integration Integrations
		dmchannels  []DmChannels
		channelIDs  []string
		thread      Threads
	)

	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error), http.StatusInternalServerError
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	exists := postgresql.CheckExists(tx, &integration, "id = ?", agentID)
	if !exists {
		tx.Rollback()
		return errors.New("agent app does not exist"), http.StatusBadRequest
	}

	err := tx.Delete(&Integrations{}, "id = ?", agentID).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete organisation integration: %w", err), http.StatusInternalServerError
	}

	err = tx.Delete(&IntegrationSettings{}, "integration_id = ?", agentID).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete custom integration settings: %w", err), http.StatusInternalServerError
	}

	err = tx.Delete(&OrganisationIntegrations{}, "integration_id = ?", agentID).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete organisation integration: %w", err), http.StatusInternalServerError
	}

	err = postgresql.SelectAllFromDb(tx, "", &dmchannels, "chat_type = 'bot' AND participant_id = ?", agentID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to fetch bot DM channels: %w", err), http.StatusInternalServerError
	}

	if len(dmchannels) > 0 {
		for _, channel := range dmchannels {
			channelIDs = append(channelIDs, channel.ChannelId)
		}

		err = postgresql.HardDeleteRecordFromDb(tx, &dmchannels)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete bot DM channels: %w", err), http.StatusInternalServerError
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err), http.StatusInternalServerError
	}

	if len(channelIDs) > 0 {
		for _, channelID := range channelIDs {
			thread.ID = channelID
			_, err := thread.ClearThreadsByChannelID(db)
			if err != nil {
				logger.Error("Warning: Failed to clear threads for channel %s: %v", channelID, err)
			}
		}
	}

	return nil, http.StatusOK
}

func (oi *OrganisationIntegrations) CheckAgentExists(db *gorm.DB, agentID string) (bool, error) {
	var agent OrganisationIntegrations

	err := db.Where("integration_id = ?", agentID).First(&agent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}