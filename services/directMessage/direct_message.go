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
		resp, err = dmchans.CreateAgentDMChannel(extReq, db, rds)
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

func GetDmParticipants(req models.DmChannelsRequest, db *gorm.DB, c *gin.Context, extReq request.ExternalRequest, rds *redis.Client) ([]gin.H, int, error) {
	var (
		user      models.User
		is_agent  bool = false
		orgAgent  models.OrganisationIntegrations
		dmchannel models.DmChannels
	)

	resp := []gin.H{}

	_, err := dmchannel.FetchChannelParticipant(db, req)
	if err != nil {
		return []gin.H{}, http.StatusBadRequest, err
	}

	if dmchannel.ChatType == "bot" {
		//check if user is a bot
		is_agent = postgresql.CheckExists(db, &orgAgent, "integration_id = ? AND org_id = ?", dmchannel.ParticipantId, req.OrgId)
		if !is_agent {
			return resp, http.StatusNotFound, fmt.Errorf("user not found: %v", err)
		}

		agentDetails, err := models.FetchDetailsFromAgentJSON(extReq, orgAgent, rds)
		if err != nil {
			return resp, http.StatusInternalServerError, fmt.Errorf("failed to fetch agent details: %w", err)
		}

		appName, appLogo := utility.GetString(agentDetails, "app_name"), utility.GetString(agentDetails, "app_logo")
		if appName == "" {
			return resp, http.StatusInternalServerError, errors.New("missing required agent details (app_name, app_logo)")
		}
		resp = append(resp, gin.H{
			"avatar_url": appLogo,
			"username":   appName,
			"email":      appName,
			"name":       appName,
			"user_type":  "bot",
			"user_id":    dmchannel.ParticipantId,
		})
		return resp, http.StatusOK, nil
	}

	if dmchannel.ChannelType == "group_dm" {

		var (
			chanPart []models.ChannelParticipant
		)

		_ = postgresql.SelectAllFromDb(db, "", &chanPart, "channel_id = ?", dmchannel.ChannelId)

		for _, part := range chanPart {
			userDetails, _ := user.GetUserByID(db, part.UserId)

			if userDetails.Profile.UserName == "" {
				userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
			}

			resp = append(resp, gin.H{
				"avatar_url": userDetails.Profile.AvatarURL,
				"username":   userDetails.Profile.UserName,
				"email":      userDetails.Email,
				"user_type":  "user",
				"user_id":    part.UserId,
			})
		}
	} else if dmchannel.ChannelType == "dm" {

		userDetails, _ := user.GetUserByID(db, *dmchannel.ParticipantId)

		if userDetails.Profile.UserName == "" {
			userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
		}

		resp = append(resp, gin.H{
			"avatar_url": userDetails.Profile.AvatarURL,
			"username":   userDetails.Profile.UserName,
			"email":      userDetails.Email,
			"user_type":  "user",
			"user_id":    dmchannel.ParticipantId,
		})

	}

	return resp, http.StatusOK, err
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
