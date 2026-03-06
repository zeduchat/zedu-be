package models

import (
	"time"

	"github.com/gofrs/uuid"
)

type AccountDeletionRequest struct {
	ID             string    `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	FullName       string    `gorm:"column:fullname; type:varchar(255)" json:"fullname"`
	Email          string    `gorm:"column:email; type:varchar(255)" json:"email"`
	Reason         string    `gorm:"column:reason; type:text" json:"reason"`
	AdditionalInfo string    `gorm:"column:additional_info; type:text" json:"additional_info"`
	OrgID          string    `gorm:"column:org_id; type:uuid" json:"org_id"`
	CreatedAt      time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
}

type CreateAccountDeletionRequest struct {
	FullName       string `json:"fullname" validate:"required"`
	Email          string `json:"email" validate:"required,email"`
	Reason         string `json:"reason" validate:"required"`
	AdditionalInfo string `json:"additional_info"`
}

func (c *CreateAccountDeletionRequest) ToAccountDeletionRequest(orgID string) (*AccountDeletionRequest, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	return &AccountDeletionRequest{
		ID:             id.String(),
		FullName:       c.FullName,
		Email:          c.Email,
		Reason:         c.Reason,
		AdditionalInfo: c.AdditionalInfo,
		OrgID:          orgID,
	}, nil
}
