package models

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

type ChannelParticipant struct {
	ID           string         `gorm:"type:uuid" json:"id"`
	ChannelId    string         `gorm:"type:uuid" json:"channel_id"`
	UserId       string         `gorm:"type:uuid" json:"user_id"`
	OrgId        string         `gorm:"type:uuid" json:"org_id"`
	CreatedAt    time.Time      `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	UpdatedAt    time.Time      `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	ThreadCount  int64          `gorm:"column:thread_count;default:0" json:"thread_count"`
	LastThreadId string         `gorm:"columnhrea:last_thread_id" json:"last_thread_id"`
	LastReadAt   time.Time      `gorm:"column:last_read_at" json:"last_read_at"`
	Title        string         `gorm:"column:title" json:"title"`
}

type GroupDMChannelsRequest struct {
	Participants []string `json:"participants" validate:"required,min=2,max=6,dive,uuid"`
	OrgId        string   `json:"org_id"`
	ChatType     string   `json:"chat_type" validate:"required,oneof=user bot"`
	UserId       string   `json:"user_id"`
	ChannelId    string   `json:"channel_id"`
}

type ParticipantInfo struct {
	UserId    string `json:"user_id"`
	Name      string `json:"username"`
	AvatarUrl string `json:"avatar_url"`
	Email     string `json:"email"`
}

type GroupDMChannelsResponse struct {
	ChannelId    string            `json:"channel_id"`
	ChannelType  string            `json:"channel_type"` // "group_dm"
	Participants []ParticipantInfo `json:"participants"` // List of all participants (excluding initiator)
}

func (dm *DmChannels) CreateGroupDMChannel(db *gorm.DB, req GroupDMChannelsRequest) (GroupDMChannelsResponse, int, error) {
	var (
		user         User
		gpdmchanresp GroupDMChannelsResponse
		existDmchan  DmChannels
		chParts      ChannelParticipant
		chPartsResp  []ChannelParticipant
		partInfo     ParticipantInfo
	)

	if len(req.Participants) < 2 {
		return gpdmchanresp, http.StatusBadRequest, fmt.Errorf("group DM channel must have at least 2 valid additional participants")
	}

	allParticipants, participantHash := utility.GenerateParticipantHash(req.Participants)

	exists := postgresql.CheckExists(db, &existDmchan, "participant_hash = ? AND org_id = ?", participantHash, req.OrgId)
	if exists {
		gpdmchanresp.ChannelId = existDmchan.ChannelId
		gpdmchanresp.ChannelType = existDmchan.ChannelType
		gpdmchanresp.Participants = []ParticipantInfo{}
		err := postgresql.SelectAllFromDb(db, "", &chPartsResp, "channel_id = ?", existDmchan.ChannelId)
		if err != nil {
			return gpdmchanresp, http.StatusInternalServerError, fmt.Errorf("failed to get participants for group DM channel %s", existDmchan.ChannelId)
		}
		for _, part := range chPartsResp {
			userDetails, err := user.GetUserByID(db, part.UserId)
			if err != nil {
				continue
			}

			if userDetails.Profile.UserName == "" {
				userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
			}
			partInfo.UserId = part.UserId
			partInfo.Name = userDetails.Profile.UserName
			partInfo.AvatarUrl = userDetails.Profile.AvatarURL
			partInfo.Email = userDetails.Email

			gpdmchanresp.Participants = append(gpdmchanresp.Participants, partInfo)
		}
		return gpdmchanresp, http.StatusOK, nil
	}

	dm.ParticipantHash = participantHash
	err := postgresql.CreateOneRecord(db, &dm)
	if err != nil {
		return gpdmchanresp, http.StatusInternalServerError, fmt.Errorf("failed to create group DM channel")
	}

	for _, participantID := range allParticipants {
		userDetails, err := user.GetUserByID(db, participantID)
		if err != nil {
			continue
		}

		if userDetails.Profile.UserName == "" {
			userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
		}

		partInfo.UserId = participantID
		partInfo.Name = userDetails.Profile.UserName
		partInfo.AvatarUrl = userDetails.Profile.AvatarURL
		partInfo.Email = userDetails.Email

		gpdmchanresp.Participants = append(gpdmchanresp.Participants, partInfo)

		// Add the participant to the channel
		chParts.ID = utility.GenerateUUID()
		chParts.ChannelId = dm.ChannelId
		chParts.UserId = participantID
		chParts.OrgId = req.OrgId
		err = postgresql.CreateOneRecord(db, &chParts)
		if err != nil {
			return gpdmchanresp, http.StatusInternalServerError, fmt.Errorf("failed to add participant %s to the channel", userDetails.Name)
		}
	}

	gpdmchanresp.ChannelId = dm.ChannelId
	gpdmchanresp.ChannelType = "group_dm"

	return gpdmchanresp, http.StatusCreated, nil
}

func (dm *DmChannels) LeaveGroupDMChannel(db *gorm.DB) (int, error) {
	var (
		user                         User
		existDM                      DmChannels
		chanPart                     ChannelParticipant
		thread                       Threads
		remainingChannelParticipants []ChannelParticipant
	)

	_, err := user.GetUserByID(db, dm.UserId)
	if err != nil {
		return http.StatusNotFound, fmt.Errorf("user does not exist")
	}

	exists := postgresql.CheckExists(db, &existDM, "channel_id = ?", dm.ChannelId)
	if !exists {
		return http.StatusNotFound, fmt.Errorf("group DM channel does not exist")
	}

	isParticipant := postgresql.CheckExists(db, &chanPart, "channel_id = ? AND user_id = ?", dm.ChannelId, dm.UserId)
	if !isParticipant {
		return http.StatusForbidden, fmt.Errorf("user is not a participant in the group DM channel")
	}

	err = postgresql.HardDeleteSpecificRecord(
		db,
		&chanPart,
		"channel_id = ? AND user_id = ?",
		dm.ChannelId,
		dm.UserId,
	)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	//fetch remainining participant and recompute the participant hash and update the dmchannel entry
	err = postgresql.SelectAllFromDb(db, "", &remainingChannelParticipants, "channel_id = ?", dm.ChannelId)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to fetch remaining channel participants %v", err)
	}

	rcp := []string{}

	for _, chap := range remainingChannelParticipants {
		rcp = append(rcp, chap.ID)
	}

	allParticipants, participantHash := utility.GenerateParticipantHash(rcp)
	update := make(map[string]any)
	update["participant_hash"] = participantHash

	result, err := postgresql.UpdateFields(db, &existDM, update, "channel_id = ?", dm.ChannelId)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to update participant hash: %v", err)
	}

	if result.RowsAffected == 0 {
		return http.StatusBadRequest, errors.New("no update occured during participant hash update")
	}

	if len(allParticipants) == 0 {

		err = postgresql.HardDeleteSpecificRecord(
			db,
			&DmChannels{},
			"channel_id = ?",
			dm.ChannelId,
		)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to delete group DM channel: %v", err)
		}

		thread.ID = dm.ChannelId

		if _, err := thread.ClearThreadsByChannelID(db); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to delete group DM channel threads: %v", err)
		}

	}

	return http.StatusOK, nil
}

func (dm *DmChannels) GetGroupDMChannels(db *gorm.DB, c *gin.Context) ([]GroupDMChannelsResponse, postgresql.PaginationResponse, error) {
	var user User

	dmchans := []DmChannels{}
	gpDMChansResp := []GroupDMChannelsResponse{}

	pagination := postgresql.GetPagination(c)

	// Define the query string to fetch DmChannels where the user is an active participant
	queryString := `
        dm_channels.org_id = ? AND dm_channels.chat_type = ? AND dm_channels.deleted_at IS NULL
        AND (
            -- For DMs: user_id matches the logged-in user
            (dm_channels.channel_type = 'dm' AND dm_channels.user_id = ?)
            OR
            -- For Group DMs: user is in channel_participants
            (dm_channels.channel_type = 'group_dm' AND EXISTS (
                SELECT 1 FROM channel_participants 
                WHERE channel_participants.channel_id = dm_channels.channel_id 
                AND channel_participants.user_id = ? 
                AND channel_participants.deleted_at IS NULL
            ))
        )
    `

	// Use SelectAllFromDbOrderByPaginated with the modified query
	paginationResp, err := postgresql.SelectAllFromDbOrderByPaginated(
		db, // No JOIN needed since we're using a subquery
		"created_at",
		"desc",
		pagination,
		&dmchans,
		queryString,
		dm.OrgId,
		"user",
		dm.UserId,
		dm.UserId,
	)

	if err != nil {
		return nil, paginationResp, err
	}

	for _, dmchan := range dmchans {

		var (
			chanPart     []ChannelParticipant
			partInfo     ParticipantInfo
			allPartsInfo []ParticipantInfo
		)

		err = postgresql.SelectAllFromDb(db, "", &chanPart, "channel_id = ?", dmchan.ChannelId)
		if err != nil {
			return nil, paginationResp, fmt.Errorf("failed to get participants for group DM channel %s", dmchan.ChannelId)
		}

		for _, part := range chanPart {
			userDetails, err := user.GetUserByID(db, part.UserId)
			if err != nil {
				return nil, paginationResp, err
			}

			if userDetails.Profile.UserName == "" {
				userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
			}
			partInfo.UserId = part.UserId
			partInfo.Name = userDetails.Profile.UserName
			partInfo.AvatarUrl = userDetails.Profile.AvatarURL
			partInfo.Email = userDetails.Email

			allPartsInfo = append(allPartsInfo, partInfo)
		}

		gpDMChansResp = append(gpDMChansResp, GroupDMChannelsResponse{
			ChannelId:    dmchan.ChannelId,
			ChannelType:  dmchan.ChannelType,
			Participants: allPartsInfo,
		})
	}

	return gpDMChansResp, paginationResp, nil
}
