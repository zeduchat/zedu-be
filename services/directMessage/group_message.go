package dm

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateGroupDMChannel(req models.GroupDMChannelsRequest, db *gorm.DB) (*models.GroupDMChannelsResponse, int, error) {
	var dmchans models.DmChannels

	dmchans.ChatType = req.ChatType
	dmchans.ChannelType = "group_dm"
	dmchans.OrgId = req.OrgId
	dmchans.UserId = req.UserId
	dmchans.ChannelId = utility.GenerateUUID()
	dmchans.ID = utility.GenerateUUID()

	req.Participants = append(req.Participants, req.UserId)

	participants := utility.ReturnUniqueIDs(req.Participants)

	req.Participants = participants

	resp, statusCode, err := dmchans.CreateGroupDMChannel(db, req)
	if err != nil {
		return nil, statusCode, err
	}

	return &resp, statusCode, nil
}

func LeaveGroupDMChannel(req models.DmChannelsRequest, db *gorm.DB) (int, error) {
	var dmchans models.DmChannels

	dmchans.ChannelId = req.ChannelId
	dmchans.UserId = req.UserId

	statusCode, err := dmchans.LeaveGroupDMChannel(db)
	if err != nil {
		return statusCode, err
	}

	return statusCode, nil
}

func JoinGroupDMChannel(req models.DmChannelsRequest, db *gorm.DB) (*models.GroupDMChannelsResponse, int, error) {
	var dmchans models.DmChannels

	dmchans.ChannelId = req.ChannelId
	dmchans.UserId = req.UserId
	dmchans.OrgId = req.OrgId

	resp, statusCode, err := dmchans.JoinGroupDMChannel(db)
	if err != nil {
		return nil, statusCode, err
	}

	return &resp, statusCode, nil
}

func GetGroupDMChannels(req models.GroupDMChannelsRequest, db *gorm.DB, c *gin.Context) ([]models.GroupDMChannelsResponse, postgresql.PaginationResponse, int, error) {

	var dmchans models.DmChannels

	dmchans.OrgId = req.OrgId
	dmchans.UserId = req.UserId

	resp, pagResp, err := dmchans.GetGroupDMChannels(db, c)

	if err != nil {
		return nil, pagResp, http.StatusInternalServerError, err
	}

	return resp, pagResp, http.StatusOK, err
}

func GetUserGroupDMs(req models.GroupDMChannelsRequest, db *gorm.DB, c *gin.Context) (gin.H, int, error) {

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
