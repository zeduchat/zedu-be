package channel

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gosimple/slug"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/thread"
	"github.com/hngprojects/telex_be/utility"
)

func CreateChannel(req models.CreateChannelsRequest, db *storage.Database, logger *utility.Logger) (models.Channels, int, error) {
	var (
		joinChannelsReq models.JoinChannelsRequest
		chans           models.Channels
		profile         models.Profile
	)

	channel := models.Channels{
		ID:             utility.GenerateUUID(),
		Name:           req.Name,
		Description:    req.Description,
		OwnerId:        req.UserId,
		OrganisationID: req.OrganisationID,
		IsPrivate:      req.IsPrivate,
		Topic:          req.Topic,
	}

	joinChannelsReq.ChannelsID = channel.ID
	joinChannelsReq.UserID = req.UserId
	joinChannelsReq.Username = req.Username

	exists := postgresql.CheckExists(db.Postgresql, &chans, "name = ? AND organisation_id = ?", req.Name, req.OrganisationID)
	if exists {
		return channel, http.StatusBadRequest, errors.New("that name is already taken by a channel, username, or user group in this organisation")
	}

	err := channel.CreateChannel(db.Postgresql)
	if err != nil {
		return channel, http.StatusInternalServerError, err
	}

	newchannel, err := channel.AddUserToChannel(db.Postgresql, joinChannelsReq)
	if err != nil {
		return newchannel, http.StatusBadRequest, err
	}

	err = profile.GetProfileByUserId(db.Postgresql, req.UserId)
	if err != nil {
		return channel, http.StatusInternalServerError, fmt.Errorf("failed to get profile: %v", err)
	}

	systemMsg := models.CreateThreadMsgReq{
		Content:    fmt.Sprintf("<p><span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span> created #%s</p><p></p>", req.UserId, profile.UserName, profile.UserName, channel.Name),
		Type:       "system",
		UserId:     req.UserId,
		ChannelsID: channel.ID,
		OrgId:      channel.OrganisationID,
		ThreadId:   utility.GenerateUUID(),
		Mentions: []models.Mention{
			{ID: req.UserId, Type: "user"},
		},
	}

	_, err = thread.SaveThreadMessage(systemMsg, db, logger)
	if err != nil {
		logger.Error("failed to save system message for channel %s", channel.Name)
	}

	logger.Info("Added system message for channel creation")

	webhook := models.Webhook{
		ID:          utility.GenerateUUID(),
		ChannelId:   channel.ID,
		OwnerId:     req.UserId,
		Status:      "active",
		WebhookName: fmt.Sprintf("%s's webhook", channel.Name),
	}

	slug := channel.ID
	webhookUrl := config.Config.App.WebhookApiUrl + fmt.Sprintf("/v1/webhooks/%s", slug)
	webhook.WebhookSlug = slug
	webhook.WebhookUrl = webhookUrl

	err = webhook.CreateWebhook(db.Postgresql)
	if err != nil {
		return newchannel, http.StatusBadRequest, err
	}

	newchannel.OwnerName = profile.UserName

	return newchannel, http.StatusOK, nil
}

func GetChannel(db *storage.Database, ids models.IDS) (models.GetChannelResp, int, error) {
	var (
		channel models.Channels
		chanReq models.ChannelInfo
	)

	chanReq.ChannelID = ids.ChannelID
	chanReq.UserID = ids.UserID

	chanresp, err := channel.GetChannelByID(db, chanReq)
	if err != nil {
		return chanresp, http.StatusBadRequest, err
	}

	err = models.UpdateUserChannelLastRead(db.Postgresql, ids.ChannelID, ids.UserID)
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

func JoinChannels(db *storage.Database, req models.JoinChannelsRequest, logger *utility.Logger) (models.Channels, int, error) {
	var (
		r models.Channels
	)

	channel, err := r.AddUserToChannel(db.Postgresql, req)

	if err != nil {
		return channel, http.StatusBadRequest, err
	}

	systemMsg := models.CreateThreadMsgReq{
		Content:    fmt.Sprintf("<p><span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span> joined #%s</p><p></p>", req.UserID, req.Username, req.Username, channel.Name),
		Type:       "system",
		UserId:     req.UserID,
		ChannelsID: channel.ID,
		OrgId:      channel.OrganisationID,
		ThreadId:   utility.GenerateUUID(),
		Mentions: []models.Mention{
			{ID: req.UserID, Type: "user"},
		},
	}

	_, err = thread.SaveThreadMessage(systemMsg, db, logger)

	if err != nil {
		logger.Error("failed to save system message for channel %s", channel.Name)
	}

	triggerNotif := models.Notification[models.TriggerNotification]
	triggerNotif.SectionType = models.ChannelsSection
	triggerNotif.ModificationDetails = &models.ModificationDetails{
		OrgId:     channel.OrganisationID,
		ChannelId: channel.ID,
		UserId:    req.UserID,
	}
	triggerNotif.Content = models.TriggerNotificationPayload{
		TriggerAction: "create",
	}
	triggerNotif.NotificationId = utility.GenerateUUID()

	centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", channel.OrganisationID, req.UserID), triggerNotif)

	return channel, http.StatusOK, nil
}

func LeaveChannels(db *storage.Database, channels_id, user_id string, logger *utility.Logger) (int, error) {
	var channel models.Channels

	ids := models.IDS{
		UserID:    user_id,
		ChannelID: channels_id,
	}

	chanResp, _, err := GetChannel(db, ids)
	if err != nil {
		return http.StatusBadRequest, errors.New("channel does not exist")
	}

	var user models.User
	userDetails, _ := user.GetUserByID(db.Postgresql, user_id)

	systemMsg := models.CreateThreadMsgReq{
		Content:    fmt.Sprintf("<p><span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span> left #%s</p><p></p>", user_id, userDetails.Profile.UserName, userDetails.Profile.UserName, chanResp.Name),
		Type:       "system",
		UserId:     user_id,
		ChannelsID: channels_id,
		OrgId:      chanResp.OrganisationID,
		ThreadId:   utility.GenerateUUID(),
		Mentions: []models.Mention{
			{ID: user_id, Type: "user"},
		},
	}

	_, err = thread.SaveThreadMessage(systemMsg, db, logger)

	if err != nil {
		logger.Error("failed to save system message for channel %s", channel.Name)
	}

	logger.Info("Added system message for user leaving the channel")

	err = channel.RemoveUserFromChannels(db.Postgresql, channels_id, user_id)
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

func DeleteChannel(db *storage.Database, channelId, userId string) (int, error) {
	var (
		r       models.Channels
		chanReq models.ChannelInfo
	)

	chanReq.ChannelID = channelId
	chanReq.UserID = userId

	channel, err := r.GetChannelByID(db, chanReq)

	if channel.Name == "general" {
		return http.StatusBadRequest, errors.New("cannot delete general channel")
	}

	if channel.OwnerId != userId {
		return http.StatusUnauthorized, errors.New("user not authorized")
	}
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = channel.Delete(db.Postgresql)
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

func UpdateChannels(db *gorm.DB, req models.UpdateChannelsRequest, ids models.IDS) (models.Channels, int, error) {
	var r models.Channels
	r.ID = ids.ChannelID

	updatedChannels, code, err := r.UpdateChannels(db, req, ids.UserID)
	if err != nil {
		return updatedChannels, code, err
	}

	return updatedChannels, code, nil
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

func AddMembersToChannel(db *storage.Database, req models.JoinChannelsRequest, logger *utility.Logger) (models.Channels, error) {
	var (
		r models.Channels
	)

	channel, err := r.AddUserToChannel(db.Postgresql, req)

	if err != nil {
		return channel, err
	}

	systemMsg := models.CreateThreadMsgReq{
		Content:    fmt.Sprintf("<p><span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span> joined #%s</p><p></p>", req.UserID, req.Username, req.Username, channel.Name),
		Type:       "system",
		UserId:     req.UserID,
		ChannelsID: channel.ID,
		OrgId:      channel.OrganisationID,
		ThreadId:   utility.GenerateUUID(),
		Mentions: []models.Mention{
			{ID: req.UserID, Type: "user"},
		},
	}

	_, err = thread.SaveThreadMessage(systemMsg, db, logger)

	if err != nil {
		logger.Error("failed to save system message for channel %s", channel.Name)
	}

	triggerNotif := models.Notification[models.TriggerNotification]
	triggerNotif.SectionType = models.ChannelsSection
	triggerNotif.ModificationDetails = &models.ModificationDetails{
		OrgId:     channel.OrganisationID,
		ChannelId: channel.ID,
		UserId:    req.UserID,
	}
	triggerNotif.Content = models.TriggerNotificationPayload{
		TriggerAction: "create",
	}
	triggerNotif.NotificationId = utility.GenerateUUID()

	centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", channel.OrganisationID, req.UserID), triggerNotif)

	return channel, nil
}

func AddMultipleMembersToChannel(db *storage.Database, req models.AddMultipleMembersRequest, logger *utility.Logger) error {
	var (
		ch   models.Channels
		user models.User
	)

	validUserIds, err := ch.AddMultipleUsersToChannel(db.Postgresql, req)
	if err != nil {
		return err
	}

	var (
		mentionList  []models.Mention
		displayNames []string
	)

	for _, userId := range validUserIds {
		userDetails, err := user.GetUserByID(db.Postgresql, userId)
		if err != nil {
			logger.Error("Failed to get user %s username", userId)
			continue
		}

		uName := userDetails.Profile.UserName
		if uName == "" {
			uName = strings.Split(userDetails.Email, "@")[0]
		}

		mentionList = append(mentionList, models.Mention{ID: userId, Type: "user"})
		displayNames = append(displayNames, fmt.Sprintf("<span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span>", userId, uName, uName))
	}

	var content string
	if len(displayNames) == 1 {
		content = fmt.Sprintf("<p>%s joined #%s</p><p></p>", displayNames[0], ch.Name)
	} else if len(displayNames) == 2 {
		content = fmt.Sprintf("<p>%s and %s joined #%s</p><p></p>", displayNames[0], displayNames[1], ch.Name)
	} else {
		lastUser := displayNames[len(displayNames)-1]
		otherUsers := strings.Join(displayNames[:len(displayNames)-1], ", ")
		content = fmt.Sprintf("<p>%s and %s joined #%s</p><p></p>", otherUsers, lastUser, ch.Name)
	}

	systemMsg := models.CreateThreadMsgReq{
		Content:    content,
		Type:       "system",
		UserId:     validUserIds[0],
		ChannelsID: ch.ID,
		OrgId:      ch.OrganisationID,
		ThreadId:   utility.GenerateUUID(),
		Mentions:   mentionList,
	}

	_, err = thread.SaveThreadMessage(systemMsg, db, logger)

	if err != nil {
		logger.Error("failed to save system message for channel %s", ch.Name)
	}

	triggerNotif := models.Notification[models.TriggerNotification]
	triggerNotif.SectionType = models.ChannelsSection
	triggerNotif.ModificationDetails = &models.ModificationDetails{
		OrgId:     ch.OrganisationID,
		ChannelId: ch.ID,
	}
	triggerNotif.Content = models.TriggerNotificationPayload{
		TriggerAction: "create",
	}
	triggerNotif.NotificationId = utility.GenerateUUID()

	for _, uID := range validUserIds {
		centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", ch.OrganisationID, uID), triggerNotif)
	}

	return nil
}

func RemoveMultipleMembersFromChannel(db *storage.Database, req models.RemoveMultipleMembersRequest, logger *utility.Logger) error {
	var (
		ch   models.Channels
		user models.User
	)

	removedUserIds, err := ch.RemoveMultipleUsersFromChannel(db.Postgresql, req)
	if err != nil {
		return err
	}

	var (
		mentionList  []models.Mention
		displayNames []string
	)

	for _, userId := range removedUserIds {
		userDetails, err := user.GetUserByID(db.Postgresql, userId)
		if err != nil {
			logger.Error("Failed to get user %s username", userId)
			continue
		}
		uName := userDetails.Profile.UserName
		if uName == "" {
			uName = strings.Split(userDetails.Email, "@")[0]
		}
		mentionList = append(mentionList, models.Mention{ID: userId, Type: "user"})
		displayNames = append(displayNames, fmt.Sprintf("<span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span>", userId, uName, uName))
	}

	if len(displayNames) == 0 {
		return nil
	}

	var content string
	if len(displayNames) == 1 {
		content = fmt.Sprintf("<p>%s left #%s</p><p></p>", displayNames[0], ch.Name)
	} else if len(displayNames) == 2 {
		content = fmt.Sprintf("<p>%s and %s left #%s</p><p></p>", displayNames[0], displayNames[1], ch.Name)
	} else {
		lastUser := displayNames[len(displayNames)-1]
		otherUsers := strings.Join(displayNames[:len(displayNames)-1], ", ")
		content = fmt.Sprintf("<p>%s and %s left #%s</p><p></p>", otherUsers, lastUser, ch.Name)
	}

	systemMsg := models.CreateThreadMsgReq{
		Content:    content,
		Type:       "system",
		UserId:     removedUserIds[0],
		ChannelsID: ch.ID,
		OrgId:      ch.OrganisationID,
		ThreadId:   utility.GenerateUUID(),
		Mentions:   mentionList,
	}

	if _, err := thread.SaveThreadMessage(systemMsg, db, logger); err != nil {
		logger.Error("failed to save system message for channel %s", ch.Name)
	}

	logger.Info("Added system message for users leaving the channel")
	return nil
}

func ArchiveChannel(db *gorm.DB, channelId string, req models.ArchiveChannelRequest) (bool, int, error) {
	var channel models.Channels

	status, err := channel.ArchiveChannel(db, channelId, req)
	if err != nil {
		return status, http.StatusBadRequest, err
	}
	return status, http.StatusOK, nil
}

func GetArchivedChannels(db *gorm.DB, ids map[string]string) ([]models.Channels, int, error) {
	var (
		channel models.Channels
		org     models.Organisation
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", ids["organisation_id"])
	if !exists {
		return nil, http.StatusBadRequest, errors.New("organisation not found")
	}

	channels, err := channel.GetArchivedChannels(db, ids)
	if err != nil {
		return channels, http.StatusBadRequest, err
	}
	return channels, http.StatusOK, nil
}

func GetUserChannels(db *storage.Database, ids models.IDS) (models.GetUserChannelResp, error) {
	var (
		uc models.UserChannels
		o  models.Organisation
	)

	_, err := o.CheckOrgExists(ids.OrganisationID, db.Postgresql)
	if err != nil {
		return nil, err
	}

	userchannels, err := uc.GetUserChannels(db, ids)
	if err != nil {
		return userchannels, err
	}

	for i := range userchannels {
		parts := strings.Split(userchannels[i].ID, "-")
		lastPart := parts[len(parts)-1]
		userchannels[i].ChannelSlug = fmt.Sprintf("%s-%s", slug.Make(userchannels[i].Name), lastPart)
	}

	return userchannels, nil
}

func GetUserNotInChannels(db *gorm.DB, ids models.IDS) (models.GetUserNotChannelResp, error) {
	var (
		uc models.UserChannels
		o  models.Organisation
	)

	_, err := o.CheckOrgExists(ids.OrganisationID, db)
	if err != nil {
		return nil, err
	}

	userchannels, err := uc.GetUserNotInChannels(db, ids)
	if err != nil {
		return userchannels, err
	}
	return userchannels, nil
}
