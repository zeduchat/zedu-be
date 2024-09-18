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
	Name                string    `gorm:"colume:name; type:varchar(255); not null;unique" json:"name"`
	JSONUrl             string    `gorm:"column:json_url; type:varchar(255);" json:"json_url"`
	AuthCredential      string    `gorm:"column:auth_credential; type:varchar(255);" json:"auth_credential"`
	IntegrationType     string    `gorm:"column:integration_type; type:varchar(255);" json:"integration_type"`
	IsSystemIntegration bool      `gorm:"column:is_system_integration; type:boolean;default:false" json:"is_system_integration"`
	CreatedAt           time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
}

type UpdateIntegration struct {
	Name            string `json:"name"`
	JSONUrl         string `json:"json_url"`
	AuthCredential  string `json:"auth_credential"`
	IntegrationType string `json:"integration_type"`
}

type ActivateChannelIntegration struct {
	Activate bool `json:"activate"`
}
type DeactivateChannelIntegration struct {
	Activate bool `json:"deactivate"`
}

type OrganisationIntegrations struct {
	ID            string    `gorm:"type:uuid;primary_key" json:"id"`
	OrgID         string    `gorm:"type:uuid;" json:"org_id"`
	IntegrationID string    `gorm:"type:uuid;" json:"integration_id"`
	IsActive      bool      `gorm:"type:boolean;default:false" json:"is_active"`
	IsArchived    bool      `gorm:"type:boolean;default:false" json:"is_archived"`
	ArchivedAt    time.Time `gorm:"index" json:"-"`
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

type IntegrationsSettings struct {
	ID                string    `gorm:"type:uuid;primary_key" json:"id"`
	OrgID             string    `gorm:"type:uuid;" json:"org_id"`
	IntegrationID     string    `gorm:"type:uuid;" json:"integration_id"`
	CurrentApiKey     string    `gorm:"type:varchar(255);" json:"current_api_key"`
	BackupEndpointUrl string    `gorm:"type:varchar(255);" json:"backup_endpoint_url"`
	IsActive          bool      `gorm:"type:boolean;default:false" json:"is_active"`
	IsArchived        bool      `gorm:"type:boolean;default:false" json:"is_archived"`
	ConnectedAt       time.Time `gorm:"column:connected_at; not null; autoCreateTime" json:"connected_at"`
	LastSyncAt        time.Time `gorm:"column:lastsync_at; autoCreateTime" json:"lastsync_at"`
	ArchivedAt        time.Time `gorm:"index" json:"-"`
	CreatedAt         time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

func (i *Integrations) CreateIntegration(db *gorm.DB, req Integrations) error {

	// exists := postgresql.CheckExists(db, i, "name = ?", req.Name)
	// if exists {
	// 	return errors.New("integration app already exists")
	// }

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

	var integrations []Integrations

	// Subquery to get all integration IDs that belong to the organization
	subQuery := db.Table("organisation_integrations").
		Select("integration_id").
		Where("org_id = ?", org_id)

	// Main query to get system integrations and organization-specific integrations
	err := db.Table("integrations").
		Where("is_system_integration = true OR id IN (?)", subQuery).
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

func (oi *OrganisationIntegrations) SetIntegrationStatus(db *gorm.DB, status string, ids map[string]string) error {

	exists := postgresql.CheckExists(db, &oi, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])
	if !exists {
		return errors.New("organisation integration does not exist")
	}

	stat := "active"

	switch oi.IsActive {
	case true:
		stat = "active"
	default:
		stat = "inactive"
	}

	if status == stat {
		return errors.New("current status is already to " + stat)
	}

	result, err := postgresql.UpdateFields(db, oi, OrganisationIntegrations{IsActive: !oi.IsActive}, "org_id = ? AND integration_id = ?", oi.OrgID, oi.IntegrationID)

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
		Where("organisation_channels_integrations.organisation_id = ?", orgID).
		Offset(offset).
		Limit(pagination.Limit).
		Find(&integrations).Error; err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	// Query to get total count of integrations for pagination
	var totalIntegrations int64
	if err := db.Table("organisation_channels_integrations").
		Joins("JOIN integrations ON organisation_channels_integrations.integration_id = integrations.id").
		Where("organisation_channels_integrations.organisation_id = ?", orgID).
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

		if oci.IsActive {
			return errors.New("integration is already active")
		} else {
			update := OrganisationChannelsIntegrations{
				IsActive: true,
			}

			if err := db.Model(&OrganisationChannelsIntegrations{}).
				Where("integration_id = ?", ids["integration_id"]).
				Updates(update).Error; err != nil {
				return nil
			}
		}

	} else {

		ociInt := OrganisationChannelsIntegrations{
			ID:            utility.GenerateUUID(),
			OrgID:         ids["organisation_id"],
			ChannelID:     ids["channel_id"],
			IntegrationID: ids["integration_id"],
			IsActive:      true,
		}

		err := ociInt.CreateOrganisationChannelIntegration(db)
		if err != nil {
			return err
		}

		return nil
	}
	return nil
}

func (oci *OrganisationChannelsIntegrations) DeactivateChannelIntegration(db *gorm.DB, req DeactivateChannelIntegration, ids map[string]string) error {

	exists := postgresql.CheckExists(db, &oci, "channel_id = ? AND org_id = ? AND integration_id = ?", ids["channel_id"], ids["organisation_id"], ids["integration_id"])

	if exists {

		if !oci.IsActive {
			return errors.New("integration is already deactivated")
		} else {
			err, _ := postgresql.SelectOneFromDb(db, &oci, "channel_id = ? AND org_id = ? AND integration_id = ?", ids["channel_id"], ids["organisation_id"], ids["integration_id"])
			if err != nil {
				return err
			}

			update := OrganisationChannelsIntegrations{
				ID:       oci.ID,
				IsActive: false,
			}

			result, err := postgresql.UpdateFields(db, &oci, update, "channel_id = ? AND org_id = ? AND integration_id = ?", ids["channel_id"], ids["organisation_id"], ids["integration_id"])

			if err != nil {
				return err
			}

			if result.RowsAffected == 0 {
				return errors.New("no record updated")
			}
		}

	} else {
		return errors.New("integration does not exist")
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
