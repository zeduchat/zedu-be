package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

type GeneralInvitation struct {
	ID             string       `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Token          string       `gorm:"column:token; type:text;null" json:"token"`
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
	Status         bool   `json:"status"`
	OrganisationID string `json:"-"`
}

type ShareableInviteResponse struct {
	InvitationLink string    `json:"invitation_link"`
	Expires_At     time.Time `json:"expires_at"`
	Created_At     time.Time `json:"created_at"`
}

func (i *GeneralInvitation) CreateShareableInvite(db *gorm.DB, req ShareableInviteRequest, created_by string) error {

	token, _ := utility.GenerateInvitationToken()

	i.ID = utility.GenerateUUID()
	i.Token = token
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

func (i *GeneralInvitation) ChangeGeneralInviteStatus(db *gorm.DB, req ChangeStatus) error {
	result := db.Model(&GeneralInvitation{}).
		Where("organisation_id = ?", req.OrganisationID).
		Update("active_status", req.Status)

	if result.Error != nil {
		return fmt.Errorf("failed to update general invitation status: %s", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("no general invitations found for this organisation")
	}

	return nil
}
