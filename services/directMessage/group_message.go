package dm

import (
	"errors"
	"net/http"

	"github.com/hngprojects/telex_be/internal/models"
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

	if utility.HasDuplicates(req.Participants) {
		return &models.GroupDMChannelsResponse{}, http.StatusBadRequest, errors.New("duplicate participants not allowed")
	}

	resp, statusCode, err := dmchans.CreateGroupDMChannel(db, req)
	if err != nil {
		return nil, statusCode, err
	}

	return &resp, statusCode , nil
}

func DeleteGroupDMChannel(req models.DmChannelsRequest, db *gorm.DB) (int, error) {
	var dmchans models.DmChannels

	dmchans.ChannelId = req.ChannelId
	dmchans.UserId = req.UserId

	statusCode, err := dmchans.DeleteGroupDMChannel(db)
	if err != nil {
		return statusCode, err
	}

	return statusCode, nil
}

