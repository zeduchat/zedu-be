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
	Participants []string `json:"participants" validate:"required,min=2,max=9,dive,uuid"`
	OrgId        string   `json:"org_id"`
	ChatType     string   `json:"chat_type" validate:"required,oneof=user bot"`
	UserId       string   `json:"user_id"`
	ChannelId    string   `json:"channel_id"`
}

type AddParticipantsRequest struct {
	UserIds   []string `json:"user_ids" validate:"required,min=1,max=9,dive,uuid"`
	ChannelId string   `json:"channel_id"`
	OrgId     string   `json:"org_id"`
}

type GroupDMChannelsResponse struct {
	ChannelId    string        `json:"channel_id"`
	ChannelType  string        `json:"channel_type"` // "group_dm"
	Participants []Participant `json:"participants"` // List of all participants (excluding initiator)
}

type AddParticipantsResponse struct {
	ChannelId      string        `json:"channel_id"`
	ChannelType    string        `json:"channel_type"`
	Participants   []Participant `json:"participants"`
	AddedCount     int               `json:"added_count"`
	SkippedCount   int               `json:"skipped_count"`
	SkippedUserIds []string          `json:"skipped_user_ids,omitempty"`
	InvalidUserIds []string          `json:"invalid_user_ids,omitempty"`
}

func (dm *DmChannels) CreateGroupDMChannel(db *gorm.DB, req GroupDMChannelsRequest) (GroupDMChannelsResponse, int, error) {
	var (
		user         User
		gpdmchanresp GroupDMChannelsResponse
		existDmchan  DmChannels
		chParts      ChannelParticipant
		chPartsResp  []ChannelParticipant
	)

	if len(req.Participants) < 2 {
		return gpdmchanresp, http.StatusBadRequest, fmt.Errorf("group DM channel must have at least 2 valid additional participants")
	}

	allParticipants, participantHash := utility.GenerateParticipantHash(req.Participants)

	exists := postgresql.CheckExists(db, &existDmchan, "participant_hash = ? AND org_id = ?", participantHash, req.OrgId)
	if exists {
		gpdmchanresp.ChannelId = existDmchan.ChannelId
		gpdmchanresp.ChannelType = existDmchan.ChannelType
		gpdmchanresp.Participants = []Participant{}
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
			gpdmchanresp.Participants = append(gpdmchanresp.Participants, Participant{
				UserId:           part.UserId,
				Username:         userDetails.Profile.UserName,
				AvatarUrl:        userDetails.Profile.AvatarURL,
				DefaultAvatarUrl: userDetails.Profile.DefaultAvatarURL,
				Email:            userDetails.Email,
				Online:           userDetails.Profile.Online,
				UserType:         "user",
				IsAdmin:          part.UserId == dm.UserId,
				Title:            userDetails.Profile.Title,
			})
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

		gpdmchanresp.Participants = append(gpdmchanresp.Participants, Participant{
			UserId:           participantID,
			Username:         userDetails.Profile.UserName,
			AvatarUrl:        userDetails.Profile.AvatarURL,
			DefaultAvatarUrl: userDetails.Profile.DefaultAvatarURL,
			Email:            userDetails.Email,
			Online:           userDetails.Profile.Online,
			UserType:         "user",
			IsAdmin:          participantID == dm.UserId,
			Title:            userDetails.Profile.Title,
		})

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
		rcp = append(rcp, chap.UserId)
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

func (dm *DmChannels) JoinGroupDMChannel(db *gorm.DB) (GroupDMChannelsResponse, int, error) {
	var (
		user                         User
		existDM                      DmChannels
		chanPart                     ChannelParticipant
		remainingChannelParticipants []ChannelParticipant
		gpdmchanresp                 GroupDMChannelsResponse
	)

	_, err := user.GetUserByID(db, dm.UserId)
	if err != nil {
		return gpdmchanresp, http.StatusNotFound, fmt.Errorf("user does not exist")
	}

	exists := postgresql.CheckExists(db, &existDM, "channel_id = ?", dm.ChannelId)
	if !exists {
		return gpdmchanresp, http.StatusNotFound, fmt.Errorf("group DM channel does not exist")
	}

	isParticipant := postgresql.CheckExists(db, &chanPart, "channel_id = ? AND user_id = ?", dm.ChannelId, dm.UserId)
	if isParticipant {
		return gpdmchanresp, http.StatusConflict, fmt.Errorf("user is already a participant in the group DM channel")
	}

	err = postgresql.SelectAllFromDb(db, "", &remainingChannelParticipants, "channel_id = ?", dm.ChannelId)
	if err != nil {
		return gpdmchanresp, http.StatusInternalServerError, fmt.Errorf("failed to fetch channel participants %v", err)
	}

	if len(remainingChannelParticipants) >= 10 {
		return gpdmchanresp, http.StatusBadRequest, fmt.Errorf("group DM channel has reached maximum capacity of 10 participants")
	}

	chanPart.ID = utility.GenerateUUID()
	chanPart.ChannelId = dm.ChannelId
	chanPart.UserId = dm.UserId
	chanPart.OrgId = dm.OrgId

	err = postgresql.CreateOneRecord(db, &chanPart)
	if err != nil {
		return gpdmchanresp, http.StatusInternalServerError, fmt.Errorf("failed to add user to group DM channel: %v", err)
	}

	err = postgresql.SelectAllFromDb(db, "", &remainingChannelParticipants, "channel_id = ?", dm.ChannelId)
	if err != nil {
		return gpdmchanresp, http.StatusInternalServerError, fmt.Errorf("failed to fetch updated channel participants %v", err)
	}

	rcp := []string{}

	for _, chap := range remainingChannelParticipants {
		rcp = append(rcp, chap.UserId)
	}

	_, participantHash := utility.GenerateParticipantHash(rcp)
	update := make(map[string]any)
	update["participant_hash"] = participantHash

	result, err := postgresql.UpdateFields(db, &existDM, update, "channel_id = ?", dm.ChannelId)
	if err != nil {
		return gpdmchanresp, http.StatusInternalServerError, fmt.Errorf("failed to update participant hash: %v", err)
	}

	if result.RowsAffected == 0 {
		return gpdmchanresp, http.StatusBadRequest, errors.New("no update occured during participant hash update")
	}

	gpdmchanresp.ChannelId = dm.ChannelId
	gpdmchanresp.ChannelType = existDM.ChannelType
	gpdmchanresp.Participants = []Participant{}

	for _, part := range remainingChannelParticipants {
		userDetails, err := user.GetUserByID(db, part.UserId)
		if err != nil {
			continue
		}

		if userDetails.Profile.UserName == "" {
			userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
		}
		gpdmchanresp.Participants = append(gpdmchanresp.Participants, Participant{
			UserId:           part.UserId,
			Username:         userDetails.Profile.UserName,
			AvatarUrl:        userDetails.Profile.AvatarURL,
			DefaultAvatarUrl: userDetails.Profile.DefaultAvatarURL,
			Email:            userDetails.Email,
			Online:           userDetails.Profile.Online,
			UserType:         "user",
			IsAdmin:          part.UserId == existDM.UserId,
			Title:            userDetails.Profile.Title,
		})
	}

	return gpdmchanresp, http.StatusOK, nil
}

func (dm *DmChannels) AddParticipantsToGroupDM(db *gorm.DB, req AddParticipantsRequest) (AddParticipantsResponse, int, error) {
	var (
		user                         User
		existDM                      DmChannels
		chanPart                     ChannelParticipant
		remainingChannelParticipants []ChannelParticipant
		addParticipantsResp          AddParticipantsResponse
	)

	exists := postgresql.CheckExists(db, &existDM, "channel_id = ?", req.ChannelId)
	if !exists {
		return addParticipantsResp, http.StatusNotFound, fmt.Errorf("group DM channel does not exist")
	}

	err := postgresql.SelectAllFromDb(db, "", &remainingChannelParticipants, "channel_id = ?", req.ChannelId)
	if err != nil {
		return addParticipantsResp, http.StatusInternalServerError, fmt.Errorf("failed to fetch channel participants %v", err)
	}

	currentCount := len(remainingChannelParticipants)
	newCount := len(req.UserIds)

	if currentCount+newCount > 10 {
		return addParticipantsResp, http.StatusBadRequest, fmt.Errorf("adding %d participants would exceed maximum capacity of 10 (current: %d)", newCount, currentCount)
	}

	existingParticipantMap := make(map[string]bool)
	for _, part := range remainingChannelParticipants {
		existingParticipantMap[part.UserId] = true
	}

	validUserIds, invalidUserIds, validationErr := ValidateUserIDs(db, req.OrgId, req.UserIds)
	if validationErr != nil {
		return addParticipantsResp, http.StatusBadRequest, validationErr
	}

	var newParticipants []string
	var skippedDuplicates []string
	for _, userId := range validUserIds {
		if existingParticipantMap[userId] {
			skippedDuplicates = append(skippedDuplicates, userId)
			continue
		}
		newParticipants = append(newParticipants, userId)
	}

	if len(newParticipants) == 0 {
		return addParticipantsResp, http.StatusBadRequest, fmt.Errorf("no new participants were added (all users are already participants or invalid)")
	}

	addedCount := 0
	for _, userId := range newParticipants {
		chanPart.ID = utility.GenerateUUID()
		chanPart.ChannelId = req.ChannelId
		chanPart.UserId = userId
		chanPart.OrgId = req.OrgId

		err = postgresql.CreateOneRecord(db, &chanPart)
		if err != nil {
			return addParticipantsResp, http.StatusInternalServerError, fmt.Errorf("failed to add participant %s to group DM channel: %v", userId, err)
		}

		addedCount++
	}

	err = postgresql.SelectAllFromDb(db, "", &remainingChannelParticipants, "channel_id = ?", req.ChannelId)
	if err != nil {
		return addParticipantsResp, http.StatusInternalServerError, fmt.Errorf("failed to fetch updated channel participants %v", err)
	}

	rcp := []string{}
	for _, chap := range remainingChannelParticipants {
		rcp = append(rcp, chap.UserId)
	}

	_, participantHash := utility.GenerateParticipantHash(rcp)
	update := make(map[string]any)
	update["participant_hash"] = participantHash

	result, err := postgresql.UpdateFields(db, &existDM, update, "channel_id = ?", req.ChannelId)
	if err != nil {
		return addParticipantsResp, http.StatusInternalServerError, fmt.Errorf("failed to update participant hash: %v", err)
	}

	if result.RowsAffected == 0 {
		return addParticipantsResp, http.StatusBadRequest, errors.New("no update occured during participant hash update")
	}

	addParticipantsResp.ChannelId = req.ChannelId
	addParticipantsResp.ChannelType = existDM.ChannelType
	addParticipantsResp.Participants = []Participant{}
	addParticipantsResp.AddedCount = addedCount
	addParticipantsResp.SkippedCount = len(skippedDuplicates) + len(invalidUserIds)
	if len(skippedDuplicates) > 0 {
		addParticipantsResp.SkippedUserIds = skippedDuplicates
	}
	if len(invalidUserIds) > 0 {
		addParticipantsResp.InvalidUserIds = invalidUserIds
	}

	for _, part := range remainingChannelParticipants {
		userDetails, err := user.GetUserByID(db, part.UserId)
		if err != nil {
			continue
		}

		if userDetails.Profile.UserName == "" {
			userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
		}
		addParticipantsResp.Participants = append(addParticipantsResp.Participants, Participant{
			UserId:           part.UserId,
			Username:         userDetails.Profile.UserName,
			AvatarUrl:        userDetails.Profile.AvatarURL,
			DefaultAvatarUrl: userDetails.Profile.DefaultAvatarURL,
			Email:            userDetails.Email,
			Online:           userDetails.Profile.Online,
			UserType:         "user",
			IsAdmin:          part.UserId == existDM.UserId,
			Title:            userDetails.Profile.Title,
		})
	}

	return addParticipantsResp, http.StatusOK, nil
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

	paginationResp, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
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
			allPartsInfo []Participant
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
			allPartsInfo = append(allPartsInfo, Participant{
				UserId:           part.UserId,
				Username:         userDetails.Profile.UserName,
				AvatarUrl:        userDetails.Profile.AvatarURL,
				DefaultAvatarUrl: userDetails.Profile.DefaultAvatarURL,
				Email:            userDetails.Email,
				Online:           userDetails.Profile.Online,
				UserType:         "user",
				IsAdmin:          part.UserId == dmchan.UserId,
				Title:            userDetails.Profile.Title,
			})
		}

		gpDMChansResp = append(gpDMChansResp, GroupDMChannelsResponse{
			ChannelId:    dmchan.ChannelId,
			ChannelType:  dmchan.ChannelType,
			Participants: allPartsInfo,
		})
	}

	return gpDMChansResp, paginationResp, nil
}

func GetGroupsInCommon(db *gorm.DB, loggedInUserId, participantId, orgId string) ([]GroupInCommon, error) {
	var commonGroups []GroupInCommon

	query := `
        SELECT DISTINCT dc.channel_id
        FROM dm_channels dc
        INNER JOIN channel_participants cp1 ON cp1.channel_id = dc.channel_id
        INNER JOIN channel_participants cp2 ON cp2.channel_id = dc.channel_id
        WHERE dc.org_id = ?
          AND dc.channel_type = 'group_dm'
          AND dc.deleted_at IS NULL
          AND cp1.user_id = ? AND cp1.deleted_at IS NULL
          AND cp2.user_id = ? AND cp2.deleted_at IS NULL
    `

	var channelIds []string
	if err := db.Raw(query, orgId, loggedInUserId, participantId).Scan(&channelIds).Error; err != nil {
		return nil, err
	}

	var user User

	for _, channelId := range channelIds {
		var chanParts []ChannelParticipant
		if err := postgresql.SelectAllFromDb(db, "", &chanParts, "channel_id = ? AND deleted_at IS NULL", channelId); err != nil {
			continue
		}

		var participantNames []string
		var avatarUrl string

		for _, part := range chanParts {
			userDetails, err := user.GetUserByID(db, part.UserId)
			if err != nil {
				continue
			}

			username := userDetails.Profile.UserName
			if username == "" {
				username = strings.Split(userDetails.Email, "@")[0]
			}
			participantNames = append(participantNames, username)

			if avatarUrl == "" {
				avatarUrl = userDetails.Profile.AvatarURL
			}
		}

		commonGroups = append(commonGroups, GroupInCommon{
			ChannelID:    channelId,
			Name:         strings.Join(participantNames, ", "),
			AvatarURL:    avatarUrl,
			Participants: participantNames,
		})
	}

	return commonGroups, nil
}
