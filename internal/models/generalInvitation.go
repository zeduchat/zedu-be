package models

import (
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

type GeneralInvitation struct {
	ID             string       `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	InviteSlug     string       `gorm:"column:invite_slug; type:text;null" json:"invite_slug"`
	ActiveStatus   bool         `gorm:"type:bool;" json:"active_status"`
	Role           string       `gorm:"type:uuid;" json:"role"`
	OrganisationID string       `gorm:"type:uuid;" json:"organisation_id"`
	Organisation   Organisation `gorm:"foreignKey:OrganisationID" json:"-"`
	InvitedBy      string       `gorm:"type:uuid" json:"invited_by"`
	CreatedAt      time.Time    `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	ExpiresAt      time.Time    `gorm:"column:expires_at; not null" json:"expires_at"`
}

type ShareableInviteRequest struct {
	OrganisationID string `json:"organisation_id" validate:"required,uuid"`
	RoleID         string `json:"role_id" validate:"required,uuid"`
}

type ChangeStatus struct {
	Status bool `json:"status" validate:"required, oneof=true false`
}

type ShareableInviteResponse struct {
	InvitationLink string    `json:"invitation_link"`
	Expires_At     time.Time `json:"expires_at"`
	Created_At     time.Time `json:"created_at"`
}

func (i *GeneralInvitation) CreateShareableInvite(db *gorm.DB, req ShareableInviteRequest, created_by string) error {

	i.ID = utility.GenerateUUID()
	i.InviteSlug = i.ID[len(i.ID)-12:]
	i.ActiveStatus = true
	i.Role = req.RoleID
	i.OrganisationID = req.OrganisationID
	i.InvitedBy = created_by
	i.ExpiresAt = time.Now().UTC().Add(48 * time.Hour)

	err := postgresql.CreateOneRecord(db, &i)
	if err != nil {
		return fmt.Errorf("failed to create integration: %w", err)
	}

	return nil
}
