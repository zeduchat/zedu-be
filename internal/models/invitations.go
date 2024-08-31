package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type Invitation struct {
	ID             string       `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Email          string       `gorm:"type:varchar(100);" json:"email"`
	Token          string       `gorm:"type:varchar(255);" json:"-"`
	Status         string       `gorm:"type:varchar(100);" json:"status"`
	Role           string       `gorm:"type:uuid;" json:"role"`
	OrganisationID string       `gorm:"type:uuid;" json:"organisation_id"`
	IsTelexUser    bool         `gorm:"type:boolean;default:false" json:"is_telex_user"`
	Organisation   Organisation `gorm:"foreignKey:OrganisationID"`
	CreatedAt      time.Time    `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	ExpiresAt      time.Time    `gorm:"column:expires_at; not null" json:"expires_at"`
}

type InvitationCreateReq struct {
	Emails         []string `json:"emails" validate:"required"`
	OrganisationID string   `json:"org_id" validate:"required,uuid"`
	RoleID           string   `json:"role_id" validate:"required,uuid"`
}

type InvitationResponse struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	OrgID          string    `json:"org_id"`
	Status         string    `json:"status"`
	InviteToken    string    `json:"invite_token"`
	IsTelexUser    bool      `json:"is_telex_user"`
	InvitationLink string    `json:"invitation_link"`
	Sent_At        time.Time `json:"sent_at"`
	Expires_At     time.Time `json:"expires_at"`
}

type ResendInvitationRequest struct {
	Emails         []string `json:"emails" validate:"required"`
	OrganisationID string   `json:"org_id" validate:"required,uuid"`
}

type VerifyInvitationLinkRequest struct {
	Token string `json:"token" validate:"required"`
}

func (i *Invitation) CreateInvitations(db *gorm.DB, invitations []Invitation) error {
	
	if len(invitations) == 0 {
		return errors.New("no invitations to save")
	}

	err := postgresql.CreateMultipleRecords(db, &invitations, len(invitations))
	if err != nil {
		return err
	}
	return nil
}

func (i *Invitation) GetInvitationsByID(db *gorm.DB, user_id string) ([]Invitation, error) {
	var invitations []Invitation

	err := postgresql.SelectAllFromDb(db.Preload("Organisation"), "", &invitations, "user_id = ?", user_id)
	if err != nil {
		return nil, err
	}
	return invitations, nil
}

func (i *Invitation) GetInvitationByID(db *gorm.DB, id string) (Invitation, error) {
	var invitation Invitation
	err, _ := postgresql.SelectOneFromDb(db, &invitation, "id = ?", id)
	if err != nil {
		return invitation, errors.New("invitation does not exist")
	}
	return invitation, nil
}

func (i *Invitation) DeleteInvitation(db *gorm.DB, id string) error {
	result := db.Delete(i, id)
	if result.RowsAffected == 0 {
		return errors.New("no record found")
	}
	return nil
}

func (i *Invitation) CheckForTelexPresence(db *gorm.DB, email string, orgID string) (map[string]string, error) {
	var (
		user  User
		ogmt  OrgUserManagement
		creds map[string]string
	)

	exists := postgresql.CheckExists(db, &user, "email = ?", email)
	if !exists {
		return creds, errors.New("user with email does not exist")
	}

	fmt.Println(user.ID, orgID)

	exists = postgresql.CheckExists(db, &ogmt, "user_id = ? AND organisation_id = ?", user.ID, orgID)
	if exists {

	}

	creds = map[string]string{
		"role":   ogmt.RoleID,
		"status": ogmt.Status,
	}
	return creds, nil
}

func (i *Invitation) CheckPendingInvitations(db *gorm.DB, email string) (Invitation, bool, error) {
	var inv Invitation

	exists := postgresql.CheckExists(db, &inv, "email = ? AND status = ?", email, "invited")
	if exists {
		return inv, true, nil
	}
	return inv, false, errors.New("no pending invitations")
}

func (i *Invitation) GetInvitationLinkByToken(db *gorm.DB, token string) (Invitation, error) {
	var invitation Invitation

	err, _ := postgresql.SelectOneFromDb(db, &invitation, "token = ?", token)
	if err != nil {
		return invitation, errors.New("token does not exist")
	}

	expired := invitation.ExpiresAt.Before(time.Now().UTC())
	if expired {
		return invitation, errors.New("invitation link has expired")
	}

	if invitation.Status == "accepted" {
		return invitation, errors.New("invitation link already accepted")
	}

	return invitation, nil
}

func (i *Invitation) UpdateInvitation(db *gorm.DB, email, status string) error {

	invites := Invitation{
		Status: status,
	}

	result, err := postgresql.UpdateFields(db, i, invites, "email = ?", email)
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("no record found")
	}

	return nil
}

func (i *Invitation) UpdateResendInvitation(db *gorm.DB, email string, expiry time.Time) error {

	invites := Invitation{
		ExpiresAt: expiry,
	}

	result, err := postgresql.UpdateFields(db, i, invites, "email = ?", email)
	if err != nil {
		return fmt.Errorf("error updating %s's invitation: %v", email, err)
	}

	if result.RowsAffected == 0 {
		return errors.New("no record found")
	}

	return nil
}
