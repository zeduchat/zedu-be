package dm

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func CreateDmChannel(req models.DmChannelsRequest, db *gorm.DB, extReq request.ExternalRequest) (*models.DmChannelsResponse, int, error) {

	var dmchans models.DmChannels

	dmchans.ChatType = req.ChatType
	dmchans.ChannelType = "dm"
	dmchans.OrgId = req.OrgId
	dmchans.UserId = req.UserId
	dmchans.ChannelId = utility.GenerateUUID()
	dmchans.ID = utility.GenerateUUID()
	dmchans.ParticipantId = &req.ParticipantId

	var resp models.DmChannelsResponse
	var err error

	if req.ChatType == "bot" {
		resp, err = dmchans.CreateAgentDMChannel(extReq, db)
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

func GetDmUser(req models.DmChannelsRequest, db *gorm.DB, c *gin.Context) (gin.H, int, error) {

	var (
		userProfile models.Profile
		user        models.User

		resp gin.H
	)

	user, err := user.GetUserByID(db, req.UserId)

	if err != nil {
		return resp, http.StatusNotFound, fmt.Errorf("user does not exist")
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
