package models

import (
	"errors"
	"math"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

type Integrations struct {
	ID                  string    `gorm:"type:uuid;primary_key" json:"id"`
	Name                string    `gorm:"colume:name; type:varchar(255); not null;unique" json:"app_name"`
	JSONUrl             string    `gorm:"column:json_url; type:varchar(255);" json:"json_url"`
	AppUrl              string    `gorm:"column:app_url; type:varchar(255);" json:"app_url"`
	AppLogo             string    `gorm:"column:app_logo; type:varchar(255);" json:"app_logo"`
	AppDescription      string    `gorm:"column:app_description; type:varchar(255);" json:"app_description"`
	IntegrationType     string    `gorm:"column:integration_type; type:varchar(255);" json:"integration_type,omitempty"`
	IsSystemIntegration bool      `gorm:"column:is_system_integration; type:boolean;" json:"is_system_integration,omitempty"`
	IsActive            bool      `gorm:"type:boolean;default:false" json:"is_active"`
	CreatedAt           time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
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

type UpdateJSONSchemaRequest struct {
	JSONSchema JSONB `gorm:"column:json_schema; type:jsonb;serializer:json" json:"json_schema"`
}

type ActivateChannelIntegration struct {
	Status bool `json:"status"`
}

type IntegrationWithSettings struct {
	OrgID         string              `json:"org_id" gorm:"column:org_id"`
	IntegrationID string              `json:"integration_id" gorm:"column:integration_id"`
	Integrations  Integrations        `json:"integration" gorm:"embedded"`
	Settings      IntegrationSettings `json:"integration_settings" gorm:"embedded"`
}

type OrganisationIntegrations struct {
	ID            string    `gorm:"type:uuid;primary_key" json:"id"`
	OrgID         string    `gorm:"type:uuid;" json:"org_id"`
	IntegrationID string    `gorm:"type:uuid;" json:"integration_id"`
	IsActive      bool      `gorm:"type:boolean;default:false" json:"is_active"`
	IsArchived    bool      `gorm:"type:boolean;default:false" json:"is_archived"`
	ArchivedAt    time.Time `gorm:"index" json:"-"`
	JSONSchema    JSONB     `gorm:"column:json_schema; type:jsonb;serializer:json" json:"json_schema"`
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
	CreatedAt     time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}




func (i *Integrations) CreateIntegration(db *gorm.DB, req Integrations) error {

	err := postgresql.CreateOneRecord(db, &i)
	if err != nil {
		return err
	}

	return nil
}

func (oi *OrganisationIntegrations) CreateOrganisationIntegration(db *gorm.DB) error {

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

func (i *Integrations) GetAllIntegrationApp(db *gorm.DB, org_id string, c *gin.Context) ([]Integrations, error) {

	var (
		integrations []Integrations
		org          Organisation
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", org_id)
	if !exists {
		return nil, errors.New("organisation not found")
	}

	subQuery := db.Table("organisation_integrations").
		Select("integration_id").
		Where("org_id = ?", org_id)

	err := db.Table("integrations AS i").
		Select(`i.id, i.name, i.app_logo, i.app_url, i.json_url, i.app_description, 
				i.is_system_integration, 
				COALESCE(oi.created_at, i.created_at) AS created_at, 
				COALESCE(oi.updated_at, i.updated_at) AS updated_at, 
				COALESCE(oi.is_active, false) AS is_active, 
				CASE 
					WHEN oi.is_active IS TRUE THEN 'active' 
					ELSE 'inactive' 
				END AS status`).
		Joins("LEFT JOIN organisation_integrations AS oi ON oi.integration_id = i.id AND oi.org_id = ?", org_id).
		Where("i.is_system_integration = TRUE OR i.id IN (?)", subQuery).
		Find(&integrations).Error
	if err != nil {
		return nil, err
	}

	return integrations, nil
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

func (oi *OrganisationIntegrations) ChangeStatus(db *gorm.DB, req ChangeIntegrationStatus, ids map[string]string) error {
	var (
		integration  Integrations
		organisation Organisation
	)

	organisationExists := postgresql.CheckExists(db, &organisation, "id = ?", ids["org_id"])
	if !organisationExists {
		return errors.New("organisation does not exist")
	}

	integrationExists := postgresql.CheckExists(db, &integration, "id = ?", ids["integration_id"])
	if !integrationExists {
		return errors.New("integration app does not exist")
	}

	orgIntegrationExists := postgresql.CheckExists(db, &oi, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])

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
		return nil
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

func (oci *OrganisationChannelsIntegrations) GetOrganisationChannelIntegrations(db *gorm.DB, channel_id, orgID string, c *gin.Context) ([]Integrations, postgresql.PaginationResponse, error) {
	var integrations []Integrations
	pagination := postgresql.GetPagination(c)

	offset := (pagination.Page - 1) * pagination.Limit

	// Query to get paginated integrations
	if err := db.Table("organisation_channels_integrations").
		Select("integrations.*").
		Joins("JOIN integrations ON organisation_channels_integrations.integration_id = integrations.id").
		Where("organisation_channels_integrations.org_id = ?", orgID).
		Offset(offset).
		Limit(pagination.Limit).
		Find(&integrations).Error; err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	// Query to get total count of integrations for pagination
	var totalIntegrations int64
	if err := db.Table("organisation_channels_integrations").
		Joins("JOIN integrations ON organisation_channels_integrations.integration_id = integrations.id").
		Where("organisation_channels_integrations.org_id = ?", orgID).
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

func (i *Integrations) PerformQueries(db *gorm.DB, channel_id string) ([]IntegrationWithSettings, error) {
	var (
		channelID = channel_id
		results   []IntegrationWithSettings
	)

	//check if channel has any integrations
	
	// Fetch active integrations with their associated settings for the given channel
	err := db.Table("organisation_channels_integrations").
		Joins("JOIN integrations ON organisation_channels_integrations.integration_id = integrations.id").
		Joins("JOIN integration_settings ON integration_settings.org_id = organisation_channels_integrations.org_id AND integration_settings.integration_id = organisation_channels_integrations.integration_id").
		Where("organisation_channels_integrations.channel_id = ? AND organisation_channels_integrations.is_active = ?", channelID, true).
		Select("integrations.id AS integration_id, integrations.*, integration_settings.*").
		Scan(&results).Error

	if err != nil {
		return []IntegrationWithSettings{}, err
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