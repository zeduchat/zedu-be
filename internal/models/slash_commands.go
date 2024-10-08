package models

import "time"

type SlashCommand struct {
    ID            string     `gorm:"type:uuid;primary_key" json:"id"`
    OrgID         string     `gorm:"type:uuid;" json:"org_id"`
    IntegrationID string     `gorm:"type:uuid;" json:"integration_id"`
    Command       string     `gorm:"column:command; type:varchar(255);" json:"command"`
    ProcessingURL string     `gorm:"column:processing_url; type:varchar(255);" json:"processing_url"`
    CreatedAt     time.Time  `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
    UpdatedAt     time.Time  `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
    DeletedAt     *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}