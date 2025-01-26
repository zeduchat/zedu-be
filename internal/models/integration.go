package models

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

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
	IsActive        bool      `gorm:"type:boolean;default:false" json:"is_active"`
	CreatedAt       time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

type UpdateIntegration struct {
	Name            string `json:"name"`
	JSONUrl         string `json:"json_url"`
	AuthCredential  string `json:"auth_credential"`
	IntegrationType string `json:"integration_type"`
}

type ChangeIntegrationStatus struct {
	Status        bool   `json:"status"`
	IntegrationID string `json:"integration_id"`
	JSONSchema    JSONB  `gorm:"column:json_schema; type:jsonb;serializer:json" json:"json_schema"`
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
	JSONUrl string `json:"json_url" validate:"required"`
}

type CustomIntegrationSettingRequest struct {
	SettingEntry    map[string]interface{} `json:"setting_entry" validate:"required"`
	SerializedEntry string                 `json:"serialized_entry"`
}

type ActivateChannelIntegration struct {
	Status bool `json:"status"`
}

type OrganisationIntegrations struct {
	ID            string    `gorm:"type:uuid;primary_key" json:"id"`
	OrgID         string    `gorm:"type:uuid;" json:"org_id"`
	IntegrationID string    `gorm:"type:uuid;" json:"integration_id"`
	IsActive      bool      `gorm:"type:boolean;default:false" json:"is_active"`
	IsArchived    bool      `gorm:"type:boolean;default:false" json:"is_archived"`
	ArchivedAt    time.Time `gorm:"index" json:"-"`
	JSONSchema    JSONB     `gorm:"column:json_schema; type:jsonb;serializer:json" json:"-"`
	JSONUrl       string    `gorm:"type:text; column:json_url;" json:"json_url"`
	CreatedAt     time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

type OrganisationChannelsIntegrations struct {
	ID            string    `gorm:"type:uuid;primary_key" json:"id"`
	OrgID         string    `gorm:"type:uuid;" json:"org_id"`
	IntegrationID string    `gorm:"type:uuid;" json:"integration_id"`
	ChannelID     string    `gorm:"type:uuid;" json:"channel_id"`
	IsActive      bool      `gorm:"type:boolean;default:false" json:"is_active"`
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

type IntegrationResp []struct {
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

	var organisation Organisation

	organisationExists := postgresql.CheckExists(db, &organisation, "id = ?", oi.OrgID)
	if !organisationExists {
		return errors.New("organisation does not exist")
	}

	exists := postgresql.CheckExists(db, &oi, "org_id = ? AND integration_id = ?", oi.OrgID, oi.IntegrationID)
	if exists {
		return errors.New("organisation integration already exists")
	}

	err := postgresql.CreateOneRecord(db, &oi)
	if err != nil {
		return err
	}

	return nil
}

func (i *Integrations) GetAllIntegrationApp(db *gorm.DB, org_id string, c *gin.Context) (IntegrationResp, error) {

	var (
		integrations IntegrationResp
		org          Organisation
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
		Find(&integrations).Error
	if err != nil {
		return nil, err
	}

	return integrations, nil
}

// Get custom integrations

func (i *OrganisationIntegrations) GetCustomIntegrationApp(db *gorm.DB, org_id string, c *gin.Context) ([]OrganisationIntegrations, postgresql.PaginationResponse, error, int) {

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
		Where("org_id = ? AND json_url != ''", org_id)

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

func (i *Integrations) UpdateIntegration(db *gorm.DB, ids map[string]string, req UpdateIntegration) (Integrations, error) {
	var integration Integrations

	exists := postgresql.CheckExists(db, &integration, "id = ?", ids["integration_id"])
	if !exists {
		return integration, errors.New("integration app does not exist")
	}

	result, err := postgresql.UpdateFields(db, &integration, req, "id = ?", integration.ID)
	if err != nil {
		return integration, errors.New("failed to update integration app")
	}
	if result.RowsAffected == 0 {
		return integration, errors.New("no record updated")
	}

	updatedIntegration := Integrations{}
	err = db.Where("id = ?", integration.ID).First(&updatedIntegration).Error
	if err != nil {
		return updatedIntegration, err
	}
	return updatedIntegration, nil
}

// Delete general integration
func (i *Integrations) DeleteIntegration(db *gorm.DB, ids map[string]string) error {
	var integration Integrations

	exists := postgresql.CheckExists(db, &integration, "id = ?", ids["integration_id"])
	if !exists {
		return errors.New("integration app does not exist")
	}

	err := db.Delete(&integration, "id = ?", ids["integration_id"]).Error
	if err != nil {
		return err
	}

	//also delete entries for the integration in the organisation integrations table
	err = db.Delete(&OrganisationIntegrations{}, "integration_id = ?", ids["integration_id"]).Error
	if err != nil {
		return err
	}

	return nil
}

// Delete Custom integration
func (i *OrganisationIntegrations) DeleteCustomIntegration(db *gorm.DB, ids map[string]string) (error, int) {
	var org_integration OrganisationIntegrations

	exists := postgresql.CheckExists(db, &org_integration, "integration_id = ?", ids["integration_id"])
	if !exists {
		return errors.New("integration app does not exist"), http.StatusBadRequest
	}

	//also delete entries for the integration in the organisation integrations table
	err := db.Delete(&OrganisationIntegrations{}, "integration_id = ?", ids["integration_id"]).Error
	if err != nil {
		return err, http.StatusInternalServerError
	}

	err = db.Delete(&CustomIntegrationsSetting{}, "integration_id = ?", ids["integration_id"]).Error
	if err != nil {
		return err, http.StatusInternalServerError
	}

	return nil, http.StatusOK
}

func (oi *OrganisationIntegrations) UpdateJSONSchema(db *gorm.DB, req UpdateJSONSchemaRequest, ids map[string]string) error {

	update := make(map[string]interface{})
	update["json_schema"] = req.JSONSchema

	result, err := postgresql.UpdateFields(db, &oi, update, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])
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

	result, err := postgresql.UpdateFields(db, &oi, update, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("no record updated")
	}

	return nil
}

func (oi *OrganisationIntegrations) ChangeStatus(db *gorm.DB, req ChangeIntegrationStatus, ids map[string]string) error {
	var (
		integration  Integrations
		organisation Organisation
		channels     []Channels
		orgchannels  []OrganisationChannelsIntegrations
	)

	organisationExists := postgresql.CheckExists(db, &organisation, "id = ?", ids["org_id"])
	if !organisationExists {
		return errors.New("organisation does not exist")
	}

	orgIntegrationExists := postgresql.CheckExists(db, &oi, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])

	integrationExists := postgresql.CheckExists(db, &integration, "id = ?", ids["integration_id"])
	
	
	if !(integrationExists || orgIntegrationExists) {
		return errors.New("integration app does not exist")
	}

	//if the integration exists but does not have an entry in the organisation integrations table, create one
	if integrationExists && !orgIntegrationExists {

		oi.ID = utility.GenerateUUID()
		oi.IsActive = req.Status
		oi.OrgID = ids["org_id"]
		oi.IntegrationID = ids["integration_id"]
		oi.JSONSchema = req.JSONSchema

		err := oi.CreateOrganisationIntegration(db)
		if err != nil {
			return err
		}

		// create an integration setting entry for it
		integrationSettings := IntegrationSettings{
			ID:             utility.GenerateUUID(),
			OrgID:          ids["org_id"],
			IntegrationID:  ids["integration_id"],
			FormFieldValue: "",
			FormFieldLabel: "",
		}

		err = integrationSettings.CreateIntegrationSettings(db)
		if err != nil {
			return err
		}

		//activate integration for all channels in the organisation
		err = postgresql.SelectAllFromDb(db, "", &channels, "organisation_id = ?", ids["org_id"])
		if err != nil {
			return err
		}

		for _, channel := range channels {
			oci := OrganisationChannelsIntegrations{
				ID:            utility.GenerateUUID(),
				OrgID:         ids["org_id"],
				ChannelID:     channel.ID,
				IntegrationID: ids["integration_id"],
				IsActive:      req.Status,
			}

			orgchannels = append(orgchannels, oci)
		}

		err = postgresql.CreateMultipleRecords(db, &orgchannels, len(orgchannels))
		if err != nil {
			return err
		}

		return nil
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
		)`, ids["org_id"], ids["integration_id"], req.Status, ids["org_id"], ids["org_id"], ids["integration_id"]).Error

	if err != nil {
		return err
	}

	//when the integration has been deactivated/activated for the integration, deactivate/activate it for all channels in the organisation
	if req.Status || !req.Status {
		err := db.Model(&OrganisationChannelsIntegrations{}).
			Where("org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"]).
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

func (oci *OrganisationChannelsIntegrations) ChangeSendBackStatus(db *gorm.DB, req ChangeIntegrationStatus, ids map[string]string) error {
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

	integrationExists := postgresql.CheckExists(db, &integration, "id = ?", ids["integration_id"])
	if !integrationExists {
		return errors.New("integration app does not exist")
	}

	orgIntegrationExists := postgresql.CheckExists(db, &oi, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])

	if !orgIntegrationExists {
		return errors.New("organisation not integrated with this app")
	}

	orgChannelIntegrationExists := postgresql.CheckExists(db, &oci, "is_active = ? AND org_id = ? AND integration_id = ? AND channel_id = ?", "true", ids["org_id"], ids["integration_id"], ids["channel_id"])

	if !orgChannelIntegrationExists {
		return errors.New("organisation not found or not active")
	}

	update := make(map[string]interface{})
	update["send_back"] = req.Status

	result, err := postgresql.UpdateFields(db, &oci, update, "org_id = ? AND integration_id = ? AND channel_id = ?", oci.OrgID, oci.IntegrationID, oci.ChannelID)

	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("integration not found or not active")
	}

	return nil
}

func (oci *OrganisationChannelsIntegrations) GetOrganisationChannelIntegrations(db *gorm.DB, channel_id, orgID string, c *gin.Context) (GetChannelIntResp, postgresql.PaginationResponse, error) {
	var integrations GetChannelIntResp
	pagination := postgresql.GetPagination(c)

	offset := (pagination.Page - 1) * pagination.Limit

	// Query to get paginated integrations
	if err := db.Table("organisation_channels_integrations AS oci").
		Select("i.id , i.name, i.json_url, i.app_url, i.app_logo, i.app_description, i.created_at, i.updated_at, i.integration_type ,oci.is_active AS is_active, oci.send_back").
		Joins("LEFT JOIN integrations AS i ON oci.integration_id = i.id").
		Where("oci.org_id = ? AND oci.channel_id = ?", orgID, channel_id).
		Offset(offset).
		Limit(pagination.Limit).
		Find(&integrations).Error; err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	// Query to get total count of integrations for pagination
	var totalIntegrations int64
	if err := db.Table("organisation_channels_integrations AS oci").
		Joins("JOIN integrations ON oci.integration_id = integrations.id").
		Where("oci.org_id = ? AND oci.channel_id = ?", orgID, channel_id).
		Count(&totalIntegrations).Error; err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(totalIntegrations) / float64(pagination.Limit)))

	// Build the pagination response
	paginationResponse := postgresql.PaginationResponse{
		CurrentPage:     pagination.Page,
		PageCount:       pagination.Limit,
		TotalPagesCount: totalPages,
	}

	return integrations, paginationResponse, nil
}

func (oci *OrganisationChannelsIntegrations) ActivateChannelIntegration(db *gorm.DB, req ActivateChannelIntegration, ids map[string]string) error {

	exists := postgresql.CheckExists(db, &oci, "channel_id = ? AND org_id = ? AND integration_id = ?", ids["channel_id"], ids["organisation_id"], ids["integration_id"])

	if exists {

		update := make(map[string]interface{})
		update["is_active"] = req.Status

		result, err := postgresql.UpdateFields(db, &oci, update, "channel_id = ? AND org_id = ? AND integration_id = ?", ids["channel_id"], ids["organisation_id"], ids["integration_id"])

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
			IntegrationID: ids["integration_id"],
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

	orgId, intId := ids["organisation_id"], ids["integration_id"]

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
	orgId, intId := ids["organisation_id"], ids["integration_id"]

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
	update["setting_entry"] = req.SerializedEntry

	result, err := postgresql.UpdateFields(db, &oi, update, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("no record updated")
	}

	return nil
}

func (i *CustomIntegrationsSetting) GetCustomIntegrationSettings(db *gorm.DB, org_id string, c *gin.Context) (CustomIntegrationsSetting, int, error) {

	var (
		org  Organisation
		resp CustomIntegrationsSetting
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", org_id)
	if !exists {
		return resp, http.StatusNotFound, errors.New("organisation not found")
	}

	exists = postgresql.CheckExists(db, &resp, "org_id = ? AND integration_id = ?", i.OrgID, i.IntegrationID)
	if !exists {
		return resp, http.StatusNotFound, errors.New("organisation not found")
	}

	return resp, http.StatusOK, nil
}
