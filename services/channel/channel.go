package channel

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/typesense/typesense-go/v2/typesense"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
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

	webhook := models.Webhook{
		ID:          utility.GenerateUUID(),
		ChannelId:   channel.ID,
		OwnerId:     userId,
		Status:      "active",
		WebhookName: fmt.Sprintf("%s's webhook", channel.Name),
	}

	slug := strings.Split(webhook.ID, "-")[4]
	webhookUrl := config.Config.App.WebhookApiUrl + fmt.Sprintf("/v1/webhooks/%s", slug)
	webhook.WebhookSlug = slug
	webhook.WebhookUrl = webhookUrl

	err = webhook.CreateWebhook(db)

	if err != nil {
		return newchannel, http.StatusBadRequest, err
	}
	return newchannel, http.StatusOK, nil
}

func GetChannels(db *gorm.DB, channelID, userId string) (models.GetChannelResp, int, error) {
	var (
		channel models.Channels
		chanReq models.ChannelInfo
	)

	chanReq.ChannelID = channelID
	chanReq.UserID = userId

	chanresp, err := channel.GetChannelsByID(db, chanReq)
	if err != nil {
		return chanresp, http.StatusBadRequest, err
	}
	return chanresp, http.StatusOK, nil
}

func GetChannelsByName(db *gorm.DB, name string) ([]models.Channels, int, error) {
	var r models.Channels

	channel, err := r.GetChannelsByName(db, name)
	if err != nil {
		return channel, http.StatusBadRequest, err
	}
	return channel, http.StatusOK, nil
}

func GetChannelsMsg(channelId, userID string, db *gorm.DB) (models.MessagesResp, int, error) {
	var c models.Channels

	resp, err := c.GetChannelsMessages(db, userID, channelId)

	if err != nil {
		return models.MessagesResp{}, http.StatusBadRequest, err
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

	_, _, err := GetChannels(db, channels_id, user_id)
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
	var (
		r       models.Channels
		chanReq models.ChannelInfo
	)

	chanReq.ChannelID = channelId
	chanReq.UserID = userId

	channel, err := r.GetChannelsByID(db, chanReq)

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

func AddMembersToChannel(db *gorm.DB, req models.JoinChannelsRequest) (models.Channels, error) {
	var ch models.Channels

	channels, err := ch.AddUserToChannels(db, req)
	if err != nil {
		return channels, err
	}
	return channels, nil
}
func AddMultipleMembersToChannel(db *gorm.DB, req models.AddMultipleMembersRequest) ([]string, error) {
	var ch models.Channels

	addError, err := ch.AddMultipleUsersToChannel(db, req)
	if err != nil {
		return addError, err
	}
	return addError, nil
}

func GetUserChannels(db *gorm.DB, userID, orgID string) ([]models.ChannelInfoResponse, error) {
	var (
		uc models.UserChannels
		o  models.Organisation
	)

	_, err := o.CheckOrgExists(orgID, db)
	if err != nil {
		return nil, err
	}

	userchannels, err := uc.GetUserChannels(db, userID, orgID)
	if err != nil {
		return userchannels, err
	}
	return userchannels, nil
}

func GetUserNotInChannels(db *gorm.DB, userID, orgID string) ([]models.ChannelInfoResponse, error) {
	var (
		uc models.UserChannels
		o models.Organisation
	)

	_, err := o.CheckOrgExists(orgID, db)
	if err != nil {
		return nil, err
	}

	userchannels, err := uc.GetUserNotInChannels(db, userID, orgID)
	if err != nil {
		return userchannels, err
	}
	return userchannels, nil
}
