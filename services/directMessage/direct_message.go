package dm

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func CreateDmChannel(req models.DmChannelsRequest, db *gorm.DB, extReq request.ExternalRequest, rds *redis.Client) (*models.DmChannelsResponse, int, error) {

	var (
		dmchans models.DmChannels
		dmfetch models.DmChannels
	)

	exists := postgresql.CheckExists(db, &dmfetch, "user_id = ? AND participant_id = ? AND org_id = ?", req.UserId, req.ParticipantId, req.OrgId)

	if !exists {
		dmchans.ChannelId = utility.GenerateUUID()
		dmchans.ID = utility.GenerateUUID()
	} else {
		dmchans.ChannelId = dmfetch.ChannelId
		dmchans.ID = dmfetch.ID
	}
	dmchans.ChatType = req.ChatType
	dmchans.ChannelType = "dm"
	dmchans.OrgId = req.OrgId
	dmchans.UserId = req.UserId
	dmchans.ParticipantId = &req.ParticipantId

	var resp models.DmChannelsResponse
	var err error

	if req.ChatType == "bot" {
		resp, err = dmchans.CreateAgentDMChannel(db)
	} else {
		resp, err = dmchans.CreateDmChannel(db)
	}
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return &resp, http.StatusCreated, nil
}

func GetDmChannels(req models.DmChannelsRequest, db *gorm.DB, c *gin.Context) ([]models.DmChannelsResponse, postgresql.PaginationResponse, int, error) {

	var dmchans models.DmChannels

	dmchans.OrgId = req.OrgId
	dmchans.UserId = req.UserId

	resp, pagResp, err := dmchans.GetDmChannels(db, c)
	if err != nil {
		return nil, pagResp, http.StatusInternalServerError, err
	}

	return resp, pagResp, http.StatusOK, err
}

func GetDmParticipants(req models.DmChannelsRequest, db *storage.Database, c *gin.Context, extReq request.ExternalRequest, rds *redis.Client, includeMedia bool) (models.DmParticipantsResponse, int, error) {
	var (
		user      models.User
		is_agent  bool = false
		orgAgent  models.OrganisationIntegrations
		dmchannel models.DmChannels
	)

	resp := models.DmParticipantsResponse{
		Participants: []map[string]any{},
	}

	_, err := dmchannel.FetchChannelParticipant(db.Postgresql, req)
	if err != nil {
		return resp, http.StatusBadRequest, err
	}

	if dmchannel.ChatType == "bot" {
		//check if user is a bot
		is_agent = postgresql.CheckExists(db.Postgresql, &orgAgent, "integration_id = ? AND org_id = ?", dmchannel.ParticipantId, req.OrgId)
		if !is_agent {
			return resp, http.StatusNotFound, fmt.Errorf("user not found: %v", err)
		}

		agentDetails, err := models.FetchDetailsFromAgentJSON(orgAgent)
		if err != nil {
			return resp, http.StatusInternalServerError, fmt.Errorf("failed to fetch agent details: %w", err)
		}

		appName, appLogo := utility.GetString(agentDetails, "app_name"), utility.GetString(agentDetails, "app_logo")
		if appName == "" {
			return resp, http.StatusInternalServerError, errors.New("missing required agent details (app_name, app_logo)")
		}
		resp.Participants = append(resp.Participants, map[string]any{
			"avatar_url": appLogo,
			"username":   appName,
			"email":      appName,
			"name":       appName,
			"user_type":  "bot",
			"user_id":    dmchannel.ParticipantId,
		})
		return resp, http.StatusOK, nil
	}

	switch dmchannel.ChannelType {
	case "group_dm":

		var (
			chanPart []models.ChannelParticipant
		)

		_ = postgresql.SelectAllFromDb(db.Postgresql, "", &chanPart, "channel_id = ?", dmchannel.ChannelId)

		for _, part := range chanPart {
			userDetails, _ := user.GetUserByID(db.Postgresql, part.UserId)

			if userDetails.Profile.UserName == "" {
				userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
			}

			resp.Participants = append(resp.Participants, map[string]any{
				"avatar_url": userDetails.Profile.AvatarURL,
				"username":   userDetails.Profile.UserName,
				"email":      userDetails.Email,
				"user_type":  "user",
				"user_id":    part.UserId,
			})
		}
	case "dm":

		userDetails, _ := user.GetUserByID(db.Postgresql, *dmchannel.ParticipantId)

		if userDetails.Profile.UserName == "" {
			userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
		}

		resp.Participants = append(resp.Participants, map[string]any{
			"avatar_url": userDetails.Profile.AvatarURL,
			"username":   userDetails.Profile.UserName,
			"email":      userDetails.Email,
			"user_type":  "user",
			"user_id":    dmchannel.ParticipantId,
		})

	}

	if includeMedia && db != nil {
		dmchannel.ChannelId = req.ChannelId
		previewMedia, _, err := dmchannel.GetPreviewMedia(db, 10)
		if err == nil {
			resp.PreviewMedia = previewMedia
		}
	}

	return resp, http.StatusOK, nil
}

func DeleteDmChannel(req models.DmChannelsRequest, db *gorm.DB) (int, error) {
	var dmchans models.DmChannels

	dmchans.ID = req.ChannelId
	dmchans.UserId = req.UserId

	err := dmchans.DeleteDmChannel(db)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func GetDmChannelMedia(req models.DmChannelMediaRequest, db *storage.Database, c *gin.Context) ([]models.File, postgresql.PaginationResponse, int, error) {
	var dmchan models.DmChannels

	// Verify user is a participant in the channel
	dmchan.ChannelId = req.ChannelId
	dmchan.UserId = req.UserId

	exists, err := dmchan.CheckChannelExists(db.Postgresql, req.ChannelId, req.UserId)
	if err != nil || !exists {
		return nil, postgresql.PaginationResponse{}, http.StatusNotFound, fmt.Errorf("channel not found or user is not a participant")
	}

	// Fetch media from Elasticsearch
	media, pagResp, err := dmchan.GetChannelMedia(db, c, req.MediaType)
	if err != nil {
		return nil, pagResp, http.StatusInternalServerError, err
	}

	return media, pagResp, http.StatusOK, nil
}
