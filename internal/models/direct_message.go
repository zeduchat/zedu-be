package models

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type DmChannels struct {
	ID            string         `gorm:"type:uuid" json:"id"`
	UserId        string         `gorm:"type:uuid" json:"-"`
	ChannelId     string         `gorm:"type:uuid" json:"channel_id"`
	OrgId         string         `gorm:"type:uuid" json:"-"`
	ParticipantId string         `gorm:"type:uuid" json:"-"`
	ChatType      string         `gorm:"type:string" json:"chat_type"`
	CreatedAt     time.Time      `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	UpdatedAt     time.Time      `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type DmChannelsResponse struct {
	ID        string `json:"channel_id"`
	Name      string `json:"username"`
	AvatarUrl string `json:"avatar_url"`
}

type DmChannelsRequest struct {
	ChatType      string `json:"chat_type" validate:"required,oneof=user bot"`
	ParticipantId string `json:"participant_id" validate:"required"`
	UserId        string `json:"user_id"`
	OrgId         string `json:"org_id"`
	ChannelId     string `json:"channel_id"`
}

func (dm *DmChannels) CreateDmChannel(db *gorm.DB) (DmChannelsResponse, error) {
	var (
		user        User
		dmchanresp  DmChannelsResponse
		existDmchan DmChannels
	)

	userDetails, err := user.GetUserByID(db, dm.ParticipantId)

	if err != nil {
		return dmchanresp, errors.New("Particpant does not exist")
	}

	exists := postgresql.CheckExists(db, &existDmchan, "user_id = ? AND participant_id = ?", dm.UserId, dm.ParticipantId)
	if exists {

		dmchanresp.AvatarUrl = userDetails.Profile.AvatarURL
		dmchanresp.Name = userDetails.Profile.UserName
		dmchanresp.ID = existDmchan.ChannelId

		return dmchanresp, nil
	}

	err = postgresql.CreateOneRecord(db, &dm)
	if err != nil {
		return dmchanresp, err
	}

	dmchanresp.AvatarUrl = userDetails.Profile.AvatarURL
	dmchanresp.Name = userDetails.Profile.UserName
	dmchanresp.ID = dm.ChannelId

	return dmchanresp, nil
}

func (dm *DmChannels) DeleteDmChannel(db *gorm.DB) error {

	var user User

	_, err := user.GetUserByID(db, dm.ParticipantId)

	if err != nil {
		return err
	}

	err = postgresql.DeleteSpecificRecord(
		db,
		&DmChannels{},
		"channel_id = ? AND user_id = ?",
		dm.ChannelId,
		dm.UserId,
	)

	if err != nil {
		return err
	}

	return nil
}

func (dm *DmChannels) GetDmChannels(db *gorm.DB, c *gin.Context) ([]DmChannelsResponse, postgresql.PaginationResponse, error) {
	var user User

	dmchans := []DmChannels{}
	dmChansResp := []DmChannelsResponse{}

	pagination := postgresql.GetPagination(c)

	paginationResp, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"created_at",
		"desc",
		pagination,
		&dmchans,
		"org_id = ? AND user_id = ? AND chat_type = ?",
		dm.OrgId,
		dm.UserId,
		"user",
	)

	if err != nil {
		return nil, paginationResp, err
	}

	for _, dmchans := range dmchans {

		userDetails, err := user.GetUserByID(db, dmchans.ParticipantId)

		if err != nil {
			return nil, paginationResp, err
		}

		dmChansResp = append(dmChansResp, DmChannelsResponse{
			ID:        dmchans.ChannelId,
			Name:      userDetails.Profile.UserName,
			AvatarUrl: userDetails.Profile.AvatarURL,
		})
	}

	if err != nil {
		return nil, paginationResp, err
	}

	return dmChansResp, paginationResp, nil

}

func (r *DmChannels) CheckChannelExists(db *gorm.DB, channelID string) (bool, error) {

	exists := postgresql.CheckExists(db, &r, "channel_id = ?", channelID)
	if !exists {
		return exists, errors.New("channel does not exist")
	}

	return exists, nil
}
