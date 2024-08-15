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
	Token          string       `gorm:"type:varchar(255);" json:"token"`
	Status         string       `gorm:"type:varchar(100);" json:"status"`
	Role           string       `gorm:"type:varchar(100);" json:"role"`
	OrganisationID string       `gorm:"type:uuid;" json:"organisation_id"`
	IsTelexUser    bool         `gorm:"type:boolean;default:false" json:"is_telex_user"`
	Organisation   Organisation `gorm:"foreignKey:OrganisationID"`
	CreatedAt      time.Time    `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	ExpiresAt      time.Time    `gorm:"column:expires_at; not null" json:"expires_at"`
}

type InvitationCreateReq struct {
	Emails         []string `json:"emails" validate:"required"`
	OrganisationID string   `json:"org_id" validate:"required,uuid"`
	Role           string   `json:"role" validate:"required"`
}

type InvitationResponse struct {
	Email          string    `json:"email"`
	OrgID          string    `json:"org_id"`
	Status         string    `json:"status"`
	InviteToken    string    `json:"invite_token"`
	IsTelexUser    bool      `json:"is_telex_user"`
	InvitationLink string    `json:"invitation_link"`
	Sent_At        time.Time `json:"sent_at"`
	Expires_At     time.Time `json:"expires_at"`
}

type VerifyInvitationLinkRequest struct {
	Token string `json:"token" validate:"required"`
}

type SendInvitationLink struct {
	Email          string `json:"email"`
	InvitationLink string `json:"invitation_link"`
}

func (i *Invitation) CreateInvitations(db *gorm.DB, invitations []Invitation) error {
	var u User

	//loop through the invitations and check is the user is a telex user
	for idx, invite := range invitations {
		exists := postgresql.CheckExists(db, &u, "email = ?", invite.Email)
		if exists {
			invitations[idx].IsTelexUser = true
		}
	}

	err := postgresql.CreateMultipleRecords(db, &invitations, len(invitations))
	if err != nil {
		return err
	}
	return nil
}

func (i *Invitation) GetInvitationsByID(db *gorm.DB, user_id string) ([]Invitation, error) {
	//get all invitations with the user_id
	var invitations []Invitation

	err := postgresql.SelectAllFromDb(db.Preload("Organisation"), "", &invitations, "user_id = ?", user_id)
	if err != nil {
		return nil, err
	}
	return invitations, nil
}

func (i *Invitation) ProcessInvitationAcceptance(db *gorm.DB, userID string) (Invitation, error) {

	return Invitation{}, errors.New("not implemented")
}

func (i *Invitation) CheckForTelexPresence(db *gorm.DB, email string, orgID string) (map[string]string, bool, error) {
	var (
		user  User
		ogmt  OrgUserManagement
		creds map[string]string
	)

	err, _ := postgresql.SelectOneFromDb(db, &user, "email = ?", email)
	if err != nil {
		return creds, false, errors.New("user with this email does not exist")
	}

	err, _ = postgresql.SelectOneFromDb(db, &ogmt, "user_id = ? AND organisation_id = ?", user.ID, orgID)
	if err != nil {
		return creds, true, errors.New("user is not part of the organisation")
	}

	creds = map[string]string{
		"role":   ogmt.RoleID,
		"status": ogmt.Status,
	}

	return creds, true, nil
}

func (i *Invitation) GetInvitationLinkByToken(db *gorm.DB, token string) (Invitation, error) {
	var invitation Invitation

	if err := db.Where("token = ? AND expires_at > ?", token, time.Now()).First(&invitation).Error; err != nil {
		return invitation, errors.New("invalid or expired token")
	}

	return invitation, nil
}

func (i *Invitation) UpdateInvitation(db *gorm.DB, email, status string) error {

	invites := Invitation{
		Status: status,
	}

	result, err := postgresql.UpdateFields(db, i,invites,"email = ?", email )
	if err != nil {
		fmt.Println(err)
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("no record found")
	}

	return nil
}
