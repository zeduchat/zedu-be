package channel

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/typesense/typesense-go/v2/typesense"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func CreateChannels(req models.CreateChannelsRequest, db *gorm.DB, userId string, typesenseDb *typesense.Client) (models.Channels, int, error) {
	var joinChannelsReq models.JoinChannelsRequest

	channel := models.Channels{
		ID:             utility.GenerateUUID(),
		Name:           req.Name,
		Description:    req.Description,
		OwnerId:        userId,
		OrganisationID: req.OrganisationID,
	}

	joinChannelsReq.ChannelsID = channel.ID
	joinChannelsReq.UserID = userId
	joinChannelsReq.Username = req.Username

	err := channel.CreateChannels(db, typesenseDb)
	if err != nil {
		return channel, http.StatusBadRequest, err
	}

	newchannel, err := channel.AddUserToChannels(db, joinChannelsReq)
	if err != nil {
		return newchannel, http.StatusBadRequest, err
	}
	return newchannel, http.StatusOK, nil
}

func GetChannels(db *gorm.DB, channelID string) (models.Channels, int, error) {
	var channel models.Channels

	channel, err := channel.GetChannelsByID(db, channelID)
	if err != nil {
		return channel, http.StatusBadRequest, err
	}
	return channel, http.StatusOK, nil
}

func GetChannelsByName(db *gorm.DB, name string) ([]models.Channels, int, error) {
	var r models.Channels

	channel, err := r.GetChannelsByName(db, name)
	if err != nil {
		return channel, http.StatusBadRequest, err
	}
	return channel, http.StatusOK, nil
}

func GetChannelsMsg(channelId, userID string, db *gorm.DB) ([]models.Message, int, error) {
	var m models.Message

	resp, err := m.GetMessagesByChannelsID(db, userID, channelId)

	if err != nil {
		return []models.Message{}, http.StatusBadRequest, err
	}

	return resp, http.StatusOK, nil

}

func JoinChannels(db *gorm.DB, req models.JoinChannelsRequest) (models.Channels, int, error) {
	var r models.Channels

	channel, err := r.AddUserToChannels(db, req)

	if err != nil {
		return channel, http.StatusBadRequest, err
	}

	return channel, http.StatusOK, nil
}

func LeaveChannels(db *gorm.DB, channels_id, user_id string) (int, error) {
	var channel models.Channels

	_, _, err := GetChannels(db, channels_id)
	if err != nil {
		return http.StatusBadRequest, errors.New("channel does not exist")
	}

	err = channel.RemoveUserFromChannels(db, channels_id, user_id)
	if err != nil {
		return http.StatusBadRequest, err
	}
	return http.StatusOK, nil

}

func UpdateUsername(req models.UpdateChannelsUserNameReq, db *gorm.DB, channelId, userId string) (int, error) {

	var ur models.UserChannels

	err := ur.UpdateUsername(db, req, channelId, userId)
	if err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}

func DeleteChannels(db *gorm.DB, channelId, userId string, typesenseDb *typesense.Client) (int, error) {
	var r models.Channels

	channel, err := r.GetChannelsByID(db, channelId)

	if channel.OwnerId != userId {
		return http.StatusUnauthorized, errors.New("user not authorized")
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = channel.Delete(db, typesenseDb)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func CountChannelsUsers(db *gorm.DB, channelId string) (int64, int, error) {
	var userChannels models.UserChannels

	count, err := userChannels.CountChannelsUsers(db, channelId)
	if err != nil {
		return count, http.StatusBadRequest, err
	}
	return count, http.StatusOK, nil
}

func UpdateChannels(db *gorm.DB, req models.UpdateChannelsRequest, channelId string, userId string) (models.Channels, error) {
	var r models.Channels
	updatedChannels, _, err := r.UpdateChannels(db, req, channelId, userId)
	if err != nil {
		return updatedChannels, err
	}
	return updatedChannels, nil
}

func CheckUser(channelId, userID string, db *gorm.DB) (gin.H, int, error) {
	var (
		userchannel models.UserChannels
		resp        gin.H
	)

	status, chk := userchannel.CheckUser(db, userID, channelId)

	resp = gin.H{
		"exist": status,
		"msg":   chk,
	}

	return resp, http.StatusOK, nil
}

func SearchChannelsByNames(db *gorm.DB, c *gin.Context, name string) ([]models.Channels, postgresql.PaginationResponse, error) {
	var (
		channel models.Channels
	)
	channels, paginationResponse, err := channel.SearchChannelssByName(db, c, name)

	if err != nil {
		return channels, paginationResponse, err
	}

	return channels, paginationResponse, nil
}

func GetUsersInChannel(channelID string, userId string, db *gorm.DB, c *gin.Context) ([]models.User, postgresql.PaginationResponse, error) {
	var channel models.Channels

	users, paginationResponse, err := channel.GetUsersInChannel(c, db, channelID)

	if err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	return users, paginationResponse, nil
}
