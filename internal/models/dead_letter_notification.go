package models

import (
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type DeadLetterNotification struct {
	ID          string      `gorm:"type:uuid;primaryKey;not null" json:"id"`
	ChannelId   string      `gorm:"type:varchar(100);index" json:"channel_id"`
	OrgId       string      `gorm:"type:varchar(100);index" json:"org_id"`
	ChannelType ChannelType `gorm:"type:varchar(50)" json:"channel_type"`
	Payload     string      `gorm:"type:jsonb;not null" json:"payload"`
	RetryCount  int         `gorm:"not null;default:0" json:"retry_count"`
	LastError   string      `gorm:"type:text" json:"last_error"`
	CreatedAt   time.Time   `gorm:"autoCreateTime;index" json:"created_at"`
}

func (d *DeadLetterNotification) Save(db *gorm.DB) error {
	return postgresql.CreateOneRecord(db, d)
}
