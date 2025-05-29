package models

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

type Integrations struct {
	ID              string    `gorm:"type:uuid;primary_key" json:"id"`
	Name            string    `gorm:"colume:name; type:varchar(255); not null;unique" json:"app_name"`
	JSONUrl         string    `gorm:"column:json_url; type:varchar(255);" json:"json_url"`
	AppUrl          string    `gorm:"column:app_url; type:varchar(255);" json:"app_url"`
	AppLogo         string    `gorm:"column:app_logo; type:varchar(255);" json:"app_logo"`
	AppDescription  string    `gorm:"column:app_description; type:varchar(255);" json:"app_description"`
	IntegrationType string    `gorm:"column:integration_type; type:varchar(255);" json:"integration_type,omitempty"`
	Info            string    `gorm:"colummn:info; type:varchar(255);" json:"info"`
	IsActive        bool      `gorm:"type:boolean;default:false" json:"is_active"`
	Category        string    `json:"category"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

type UpdateAgent struct {
	Name            string `json:"name"`
	JSONUrl         string `json:"json_url"`
	AuthCredential  string `json:"auth_credential"`
	IntegrationType string `json:"integration_type"`
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
	SettingEntry    map[string]interface{} `json:"setting_entry" validate:"required"`
	SerializedEntry string                 `json:"serialized_entry"`
}

type ActivateChannelAgent struct {
	Status bool `json:"status"`
}

type OrganisationIntegrations struct {
	ID             string    `gorm:"type:uuid;primary_key" json:"id"`
	OrgID          string    `gorm:"type:uuid;" json:"org_id"`
	IntegrationID  string    `gorm:"type:uuid;" json:"integration_id"`
	IsActive       bool      `gorm:"type:boolean;default:false" json:"is_active"`
	IsSystem       bool      `gorm:"type:boolean;default:false" json:"is_system"`
	IsArchived     bool      `gorm:"type:boolean;default:false" json:"is_archived"`
	ArchivedAt     time.Time `gorm:"index" json:"-"`
	JSONSchema     JSONB     `gorm:"column:json_schema; type:jsonb;serializer:json" json:"-"`
	JSONUrl        string    `gorm:"type:text; column:json_url;" json:"json_url"`
	CreatedAt      time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	AppDescription string    `gorm:"column:app_description;type:text;" json:"app_description"`
	AppName        string    `gorm:"column:app_name;type:text;" json:"app_name"`
	AppLogo        string    `gorm:"column:app_logo;type:text;" json:"app_logo"`
	AppUrl         string    `gorm:"column:app_url; type:text;" json:"app_url"`
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
	Integrations
	Linked bool `json:"linked"`
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
				i.is_system_integration, 
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
func (i *OrganisationIntegrations) GetCustomAgentApp(db *gorm.DB, org_id string, c *gin.Context) ([]OrganisationIntegrations, postgresql.PaginationResponse, error, int) {

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
		Where("org_id = ? AND json_url != '' ", org_id)

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

	query := db.Model(&Integrations{}).
		Where("json_url != ''")

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

	exists := postgresql.CheckExists(db, &IntResp, "json_url != '' AND id = ?", int_id)
	if !exists {
		return IntResp, errors.New("integration app does not exist"), http.StatusNotFound
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
			_, err := thread.ClearDMThreadsByChannelID(db)
			if err != nil {
				logger.Error("Warning: Failed to clear threads for channel %s: %v", channelID, err)
			}
		}
	}

	return nil, http.StatusOK
}

func (oi *OrganisationIntegrations) UpdateJSONSchema(db *gorm.DB, req UpdateJSONSchemaRequest, ids map[string]string) error {

	update := make(map[string]interface{})
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

func (oi *OrganisationIntegrations) UpdateCustomIntegration(db *gorm.DB, req CustomIntegrationRequest, ids map[string]string) error {

	update := make(map[string]interface{})
	update["json_url"] = req.JSONUrl
	update["app_name"] = req.AppName
	update["app_description"] = req.AppDescription
	update["app_url"] = req.AppUrl
	update["app_logo"] = req.AppLogo

	result, err := postgresql.UpdateFields(db, &oi, update, "integration_id = ?", ids["agent_id"])
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("no record updated")
	}

	return nil
}

func (oi *OrganisationIntegrations) ChangeStatus(db *gorm.DB, req ChangeAgentStatus, ids map[string]string, extReq request.ExternalRequest) error {
	var (
		agent        Integrations
		organisation Organisation
		oci          OrganisationChannelsIntegrations
		channels     []Channels
		orgchannels  []OrganisationChannelsIntegrations
	)

	organisationExists := postgresql.CheckExists(db, &organisation, "id = ?", ids["org_id"])
	if !organisationExists {
		return errors.New("organisation does not exist")
	}

	orgAgentExists := postgresql.CheckExists(db, &oi, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	agentExists := postgresql.CheckExists(db, &agent, "id = ?", ids["agent_id"])
	ChannelagentExists := postgresql.CheckExists(db, &oci, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	// CheckIntegrationSettings := postgresql.CheckExists(db, &intsettings, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])

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

		if agentExists {
			oi.IsSystem = true
		} else {
			oi.IsSystem = false
		}

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

	// return nil

	// add settings if not exist
	// if !CheckIntegrationSettings && agent.JSONUrl != "" {
	// 	data := map[string]string{"url": agent.JSONUrl}

	// 	response, err := extReq.SendExternalRequest(request.AgentJsonContent, data)

	// 	if err != nil {
	// 		return errors.New("failed to save agent default settings, invalid JSON supplied")
	// 	}

	// 	response_data := response.(map[string]interface{})
	// 	data_r, ok := response_data["data"].(map[string]interface{})

	// 	if !ok {
	// 		return errors.New("Failed to save agent, data field does not exist")
	// 	}

	// 	// validate all entries
	// 	err = ValidateAgentData(data_r)

	// 	if err != nil {
	// 		return err
	// 	}

	// 	settings, ok := data_r["settings"]
	// 	if !ok {
	// 		return errors.New("Failed to save agent default settings, settings field does not exist")
	// 	}

	// 	settings_data := map[string]interface{}{"settings": settings}

	// 	is_auth, ok := data_r["is_oauth"].(bool)

	// 	if ok && is_auth {
	// 		enc_key := config.Config.Server.EncKey

	// 		auth_credentials := map[string]interface{}{"agent_auth_credentials": "Not-Set-Yet"}

	// 		api_key, err := utility.CreateExternalApiKey(ids["org_id"], ids["agent_id"], enc_key)

	// 		auth_credentials["telex_api_key"] = api_key
	// 		settings_data["auth_credentials"] = auth_credentials
	// 		if err != nil {
	// 			return errors.New("Failed to create external API key")
	// 		}
	// 	}

	// 	// serialize the settings json

	// 	settingJsonData, err := json.Marshal(settings_data)
	// 	if err != nil {
	// 		return fmt.Errorf("error serializing to JSON: %v", err)
	// 	}
	// 	serialized_settings := string(settingJsonData)

	// 	integrationSettings.ID = utility.GenerateUUID()
	// 	integrationSettings.SettingEntry = serialized_settings
	// 	integrationSettings.OrgID = ids["org_id"]
	// 	integrationSettings.IntegrationID = ids["agent_id"]

	// 	if agentExists {
	// 		integrationSettings.IsSystem = true
	// 	} else {
	// 		integrationSettings.IsSystem = false
	// 	}

	// 	err = integrationSettings.CreateIntegrationSettings(db)
	// 	if err != nil {
	// 		return errors.New("failed to create agent settings")
	// 	}
	// }

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

	update := make(map[string]interface{})
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

	update := make(map[string]interface{})
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

		update := make(map[string]interface{})
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

	update := make(map[string]interface{})

	deserialize_settings := make(map[string]interface{})

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

		encoded_auth_cred, ok := auth_creds.(map[string]interface{})["integration_auth_credentials"].(string)

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

func ValidateAgentData(data_r map[string]interface{}) error {

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

	_, ok = skills.([]interface{})
	if !ok {
		return errors.New("Failed to save agent, skills field is not an array")
	}

	defaultInputModes, ok := data_r["defaultInputModes"]
	if !ok {
		return errors.New("Failed to save agent, defaultInputModes field does not exist or is empty")
	}

	_, ok = defaultInputModes.([]interface{})
	if !ok {
		return errors.New("Failed to save agent, defaultInputModes field is not an array")
	}

	defaultOutputModes, ok := data_r["defaultOutputModes"]
	if !ok {
		return errors.New("Failed to save agent, defaultOutputModes field does not exist or is empty")
	}
	_, ok = defaultOutputModes.([]interface{})
	if !ok {
		return errors.New("Failed to save agent, defaultOutputModes field is not an array")
	}

	_, ok = data_r["provider"].(map[string]interface{})
	if !ok {
		return errors.New("Failed to save agent, invalid agent card: provider does not exist or is empty")
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
