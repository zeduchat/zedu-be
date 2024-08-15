package models

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type Integrations struct {
	ID             string 		`gorm:"type:uuid;primary_key" json:"id"`
	Name           string 		`gorm:"not null;unique" json:"name"`
	ApiEndpointUrl string		`gorm:"column:api_endpoint_url; type:varchar(255);" json:"api_endpoint_url"`
	AuthCredential string		`gorm:"column:auth_credential; type:varchar(255);" json:"auth_credential"`
	ConnectedAt    time.Time	`gorm:"column:connected_at; not null; autoCreateTime" json:"connected_at"`
	LastSyncAt     time.Time	`gorm:"column:lastsync_at; autoCreateTime" json:"lastsync_at"`
	CreatedAt      time.Time	`gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	ArchivedAt     time.Time	`gorm:"index" json:"-"`
}

type IntegrationsSettings struct {
	ID                      string 		`gorm:"type:uuid;primary_key" json:"id"`
	OrgID		            string      `gorm:"type:uuid;" json:"org_id"`
	IntegrationID		    string      `gorm:"type:uuid;" json:"integration_id"`
	CurrentApiKey		    string      `gorm:"type:varchar(255);" json:"current_api_key"`
	BackupEndpoitUrl		string      `gorm:"type:varchar(255);" json:"backup_endpoint_url"`
	Status		            string      `gorm:"type:varchar(255);" json:"status"` //enum
	CreatedAt 				time.Time   `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt  				time.Time   `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

func (i *Integrations) CreateIntegrationApp(db *gorm.DB, name string) error {
	var integrationApp Integrations

	exists := postgresql.CheckExists(db, &integrationApp, "name = ?", name)

	if exists {
		return errors.New("help center category already exists")
	}
	err := postgresql.CreateOneRecord(db, &i)

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


