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

	exists := postgresql.CheckExists(db, &dmfetch, "user_id = ? AND participant_id = ?", req.UserId, req.ParticipantId)

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

func GetDmUser(req models.DmChannelsRequest, db *gorm.DB, c *gin.Context, extReq request.ExternalRequest, rds *redis.Client) (gin.H, int, error) {

	var (
		userProfile models.Profile
		user        models.User
		is_agent    bool = false
		resp        gin.H
		orgAgent    models.OrganisationIntegrations
	)

	user, err := user.GetUserByID(db, req.UserId)
	if err != nil {
		//check if user is a bot
		is_agent = postgresql.CheckExists(db, &orgAgent, "integration_id = ? AND org_id = ?", req.UserId, req.OrgId)
		if !is_agent {
			return resp, http.StatusNotFound, fmt.Errorf("user not found: %v", err)
		}

		agentDetails, err := models.FetchDetailsFromAgentJSON(extReq, orgAgent.JSONUrl, rds)
		if err != nil {
			return resp, http.StatusInternalServerError, fmt.Errorf("failed to fetch agent details: %w", err)
		}
		agentDescription, ok := agentDetails["descriptions"].(map[string]interface{})
		if !ok {
			return resp, http.StatusInternalServerError, errors.New("invalid agent details format")
		}
		appName, appLogo := utility.GetString(agentDescription, "app_name"), utility.GetString(agentDescription, "app_logo")
		if appName == "" || appLogo == "" {
			return resp, http.StatusInternalServerError, errors.New("missing required agent details (app_name, app_logo)")
		}
		resp = gin.H{
			"avatar_url": appLogo,
			"username":   appName,
			"email":      appName,
			"name":       appName,
		}
		return resp, http.StatusOK, nil
	}

	err = userProfile.GetProfileByUserId(db, req.UserId)

	if err != nil {
		return resp, http.StatusInternalServerError, err
	}

	resp = gin.H{
		"avatar_url": userProfile.AvatarURL,
		"username":   userProfile.UserName,
		"email":      user.Email,
	}

	if resp["username"] == "" {
		resp["username"] = strings.Split(user.Email, "@")[0]
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
