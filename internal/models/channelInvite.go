package models

import (
	"errors"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type ChannelInvitation struct {
	ID             string       `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Email          string       `gorm:"type:varchar(100);" json:"email"`
	Token          string       `gorm:"type:varchar(255);" json:"token"`
	Status         string       `gorm:"type:varchar(100);" json:"status"`
	Role           string       `gorm:"type:varchar(100);" json:"role"`
	OrganisationID string       `gorm:"type:uuid;" json:"organisation_id"`
	ChannelID      string       `gorm:"type:uuid;" json:"channel_id"`
	Organisation   Organisation `gorm:"foreignKey:OrganisationID"`
	CreatedAt      time.Time    `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	ExpiresAt      time.Time    `gorm:"column:expires_at; not null" json:"expires_at"`
}

type ChannelInvitationCreateReq struct {
	Emails         []string `json:"emails" validate:"required"`
	OrganisationID string   `json:"organisation_id" validate:"required,uuid"`
	ChannelID      string   `json:"channel_id" validate:"required,uuid"`
	Role           string   `json:"role" validate:"required"`
}

type ChannelInvitationResponse struct {
	Email          string    `json:"email"`
	ChannelID      string    `json:"channel_id"`
	OrgID          string    `json:"org_id"`
	Status         string    `json:"status"`
	InviteToken    string    `json:"invite_token"`
	InvitationLink string    `json:"invitation_link"`
	Sent_At        time.Time `json:"sent_at"`
	Expires_At     time.Time `json:"expires_at"`
}

type ChannelVerifyInvitationLinkRequest struct {
	Token string `json:"token" validate:"required"`
}

type ChannelSendInvitationLink struct {
	Email          string `json:"email"`
	InvitationLink string `json:"invitation_link"`
}

func (c *ChannelInvitation) CreateChannelInvitations(db *gorm.DB, channelInvitations []ChannelInvitation) error {

	err := postgresql.CreateMultipleRecords(db, &channelInvitations, len(channelInvitations))
	if err != nil {
		return err
	}
	return nil
}

func (i *ChannelInvitation) GetMagicLinkByEmail(db *gorm.DB, email string) (*ChannelInvitation, error) {
	var channelInvitation ChannelInvitation

	err := db.Where("email = ?", email).First(&channelInvitation).Error
	if err != nil {
		return nil, err
	}
	return &channelInvitation, nil
}

func (i *ChannelInvitation) DeleteChannelInviteLink(db *gorm.DB) error {
	err := postgresql.DeleteRecordFromDb(db, i)
	if err != nil {
		return err
	}
	return nil
}

func (i *ChannelInvitation) CheckForChannelPresence(db *gorm.DB, email string, channelID string) error {
	var (
		user     User
		userChan UserChannels
	)
	err, _ := postgresql.SelectOneFromDb(db, &user, "email = ?", email)
	if err != nil {
		return errors.New("user with this email does not exist")
	}

	exists := postgresql.CheckExists(db, &userChan, "user_id = ? AND channels_id = ?", user.ID, channelID)

	if exists {
		return errors.New("user already exists in the channel")
	}
	return nil
}