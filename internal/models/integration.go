package models

import (
	"errors"
	"math"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type Integrations struct {
	ID                  string    `gorm:"type:uuid;primary_key" json:"id"`
	Name                string    `gorm:"colume:name; type:varchar(255); not null;unique" json:"name"`
	LogoUrl             string    `gorm:"colume:logo_url; type:varchar(255);" json:"logo_url"`
	ApiEndpointUrl      string    `gorm:"column:api_endpoint_url; type:varchar(255);" json:"api_endpoint_url"`
	AuthCredential      string    `gorm:"column:auth_credential; type:varchar(255);" json:"auth_credential"`
	IsSystemIntegration bool      `gorm:"column:is_system_integrations; type:boolean;default:false" json:"is_system_integration"`
	CreatedAt           time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
}

type UpdateIntegration struct {
	Name           string `json:"name"`
	LogoUrl        string `json:"logo_url"`
	ApiEndpoint    string `json:"api_endpoint"`
	AuthCredential string `json:"auth_credential"`
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

	exists := postgresql.CheckExists(db, i, "name = ?", req.Name)
	if exists {
		return errors.New("integration app already exists")
	}

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

func (i *Integrations) GetAllIntegrationApp(db *gorm.DB, c *gin.Context) ([]Integrations, postgresql.PaginationResponse, error) {
	var integrationApp []Integrations

	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"created_at",
		"desc",
		pagination,
		&integrationApp,
		nil,
	)

	if err != nil {
		return nil, paginationResponse, err
	}

	return integrationApp, paginationResponse, nil
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

func (oci *OrganisationChannelsIntegrations) GetOrganisationChannelIntegrations(db *gorm.DB, channel_id, orgID string,c *gin.Context) ([]Integrations, postgresql.PaginationResponse, error) {
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
