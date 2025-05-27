package models

import (
	"time"
)

type CreditUsage struct {
	ID             string    `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	OrganisationID string    `gorm:"type:uuid;not null;index" json:"organisation_id"`
	Amount         float64   `gorm:"type:decimal(10,2);not null" json:"amount"`
	AgentID        string    `gorm:"type:uuid;not null;index" json:"agent_id"`
	UserID         string    `gorm:"type:uuid;not null;index" json:"user_id"`
	CreatedAt      time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

type CreditTransaction struct {
	ID             string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganisationID string    `gorm:"type:uuid;not null;index" json:"organisation_id"`
	Amount         float64   `gorm:"type:decimal(10,2);not null" json:"amount"`
	BalanceBefore  float64   `gorm:"type:decimal(10,2);not null" json:"balance_before"`
	BalanceAfter   float64   `gorm:"type:decimal(10,2);not null" json:"balance_after"`
	Type           string    `gorm:"type:varchar(50);not null" json:"type"` // e.g., "topup", "refund"
	CreatedAt      time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}
