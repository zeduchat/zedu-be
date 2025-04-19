package models

import (
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
	ID        string         `gorm:"type:uuid" json:"id"`
	ChannelId string         `gorm:"type:uuid" json:"channel_id"`
	UserId    string         `gorm:"type:uuid" json:"user_id"`
	OrgId     string         `gorm:"type:uuid" json:"org_id"`
	CreatedAt time.Time      `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	UpdatedAt time.Time      `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
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

func (dm *DmChannels) DeleteGroupDMChannel(db *gorm.DB) (int, error) {
	var (
		user      User
		existDM   DmChannels
		chanParts ChannelParticipant
		thread    Threads
	)

	_, err := user.GetUserByID(db, dm.UserId)
	if err != nil {
		return http.StatusNotFound, fmt.Errorf("user does not exist")
	}

	exists := postgresql.CheckExists(db, &existDM, "channel_id = ?", dm.ChannelId)
	if !exists {
		return http.StatusNotFound, fmt.Errorf("group DM channel does not exist")
	}

	isParticipant := postgresql.CheckExists(db, &chanParts, "channel_id = ? AND user_id = ?", dm.ChannelId, dm.UserId)
	if !isParticipant {
		return http.StatusForbidden, fmt.Errorf("user is not a participant in the group DM channel")
	}

	if existDM.UserId == dm.UserId {
		// Perform a hard delete of the group DM channel since the initiator is deleting it
		err = postgresql.HardDeleteSpecificRecord(
			db,
			&DmChannels{},
			"channel_id = ?",
			dm.ChannelId,
		)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to delete group DM channel: %v", err)
		}

		err = postgresql.HardDeleteSpecificRecord(
			db,
			&chanParts,
			"channel_id = ?",
			dm.ChannelId,
		)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to delete group DM channel participants: %v", err)
		}

		thread.ID = dm.ChannelId

		if _, err := thread.DeleteThread(db); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to delete group DM channel threads: %v", err)
		}

		return http.StatusOK, nil
	}

	err = postgresql.HardDeleteSpecificRecord(
		db,
		&chanParts,
		"channel_id = ? AND user_id = ?",
		dm.ChannelId,
		dm.UserId,
	)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	count, err := postgresql.CountSpecificRecords(db, &chanParts, "channel_id = ?", dm.ChannelId)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to count group DM channel participants: %v", err)
	}

	if count == 0 {
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

		if _, err := thread.DeleteThread(db); err != nil {
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

	paginationResp, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"created_at",
		"desc",
		pagination,
		&dmchans,
		"org_id = ? AND user_id = ? AND chat_type = ? AND channel_type = ?",
		dm.OrgId,
		dm.UserId,
		"user",
		"group_dm",
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
