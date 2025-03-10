package dm

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func CreateDmChannel(req models.DmChannelsRequest, db *gorm.DB) (*models.DmChannelsResponse, int, error) {

	var dmchans models.DmChannels

	dmchans.ChatType = req.ChatType
	dmchans.OrgId = req.OrgId
	dmchans.UserId = req.UserId
	dmchans.ParticipantId = req.ParticipantId
	dmchans.ChannelId = utility.GenerateUUID()
	dmchans.ID = utility.GenerateUUID()

	resp, err := dmchans.CreateDmChannel(db)

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
