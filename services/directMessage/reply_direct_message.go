package dm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gofrs/uuid"

	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/thread"
	"github.com/hngprojects/telex_be/utility"
)

// Reply message fn (in dm / group_dm)
func ReplyChannelDMMessage(req models.CreateMessageRequest, db *storage.Database, logger *utility.Logger) (*models.MessageDocument, int, error) {
	var (
		profile models.Profile
		user    models.User
		channel models.DmChannels
		threads models.ThreadDocument
	)

	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("invalid thread ID")
	}

	ch, err := channel.CheckChannelExists(db.Postgresql, req.ChannelsId, req.UserId)
	if !ch || err != nil {
		return nil, http.StatusNotFound, errors.New("channel does not exist")
	}

	if req.OrgId == "" {
		req.OrgId = channel.OrgId
	}

	if err := threads.GetThreadById(req.ThreadId); err != nil {
		logger.Error(fmt.Sprintf("Failed to get thread by ID: %s, error: %v", req.ThreadId, err))
	}

	profile, err = profile.GetProfileByUserIdAndOrgId(db.Postgresql, req.UserId, channel.OrgId)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to get user profile")
	}

	user, err = user.GetUserByID(db.Postgresql, req.UserId, channel.OrgId)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to get user")
	}

	defaultAvatarURL := avatar.GenerateDefaultAvatarURL(req.UserId)

	messageDoc := models.MessageDocument{
		ID:               utility.GenerateUUID(),
		ProfileID:        profile.ID,
		Content:          req.Content,
		ChannelsID:       req.ChannelsId,
		UserID:           req.UserId,
		ThreadID:         threadId,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		AvatarURL:        profile.AvatarURL,
		DefaultAvatarURL: defaultAvatarURL,
		Edited:           false,
		Username:         profile.UserName,
		FullName:         profile.FullName,
		Email:            user.Email,
		UserType:         "user",
		OrganisationID:   channel.OrgId,
		Mentions:         req.Mentions,
		Media:            req.Media,
	}

	updateResp, err := messageDoc.CreateMessage(db, logger)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to save message, error: " + err.Error())
	}

	if err := thread.DetectAndAddMentions(messageDoc.ID, req.Content, db.Postgresql); err != nil {
		return &messageDoc, http.StatusBadRequest, err
	}

	username := utility.ThisOrThat(profile.UserName, utility.ThisOrThat(profile.FullName, user.Email))

	feed := models.FeedMessageRequest{
		ChannelID:        req.ChannelsId,
		UserName:         profile.UserName,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:        messageDoc.UpdatedAt.Format(time.RFC3339),
		AvatarURL:        profile.AvatarURL,
		DefaultAvatarUrl: defaultAvatarURL,
		Type:             "message",
		Content:          req.Content,
		ThreadId:         req.ThreadId,
		Email:            user.Email,
		FullName:         profile.FullName,
		OrgId:            channel.OrgId,
		UserType:         "user",
		UserId:           req.UserId,
		Media:            req.Media,
		Mentions:         req.Mentions,
		Id:               messageDoc.ID,
		ChannelName:      username,
		ChannelType:      channel.ChannelType,
	}

	err = centrifuge.PublishChannel(logger, threadId.String(), feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to threadId: %s, error: %v", threadId.String(), err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to publish webhook data: " + err.Error())
	}

	notification := models.Notification[models.ReplyCountChange]
	notification.SectionType = models.ChannelsSection
	notification.Content = feed
	notification.UpdateChange = updateResp

	err = centrifuge.PublishChannel(logger, req.ChannelsId, notification)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing update reply message with destination id: %s error: %v", req.ChannelsId, err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to publish data: " + err.Error())
	}

	if threads.UserId != "" && threads.OrganisationID != "" {
		threadReplyNotif := models.Notification[models.ThreadReply]
		threadReplyNotif.Content = feed
		threadReplyNotif.NotificationId = utility.GenerateUUID()

		err = centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", threads.OrganisationID, threads.UserId), threadReplyNotif)
		if err != nil {
			logger.Error(fmt.Sprintf("Error Publishing thread reply message to user, channelid: %s, with userid: %s error: %v", req.ChannelsId, threads.UserId, err.Error()))
		} else {
			logger.Info("Published thread reply message to user : %s", threads.UserId)
		}
	}

	dataByte, _ := json.Marshal(feed)

	notifRec := models.PushNotificationRecord{
		ChannelType:  models.DMChannel,
		Data:         string(dataByte),
		Sent:         false,
		ChannelId:    req.ChannelsId,
		Section:      models.ReplySection,
		UpdateChange: updateResp,
		Type:         models.NewMessage,
	}

	err = actions.AddPushNotificationToQueue(storage.DB.Redis, notifRec)

	if err != nil {
		logger.Error("Error adding notification to channelid: %s, with orgid: %s error: %v", req.ChannelsId, req.OrgId, err.Error())
	}

	logger.Info("added notification to queue for channel %s", req.ChannelsId)

	if threads.ID == "" {
		threads.ID = req.ThreadId
		threads.OrganisationID = channel.OrgId
	}

	thread.TrackThreadNotification(req.UserId, req.ChannelsId, channel.OrgId, &threads, feed, logger)

	dmChanVisibility := models.DmChannels{ChannelId: req.ChannelsId}
	_ = dmChanVisibility.UpdateInteractionAt(db.Postgresql)

	return &messageDoc, http.StatusCreated, nil
}

// Reply message fn
func AddChannelsDmMsg(req models.CreateMessageRequest, db *storage.Database,
	logger *utility.Logger) (*models.MessageDocument, int, error) {

	// Provision for bot dms

	return ReplyChannelDMMessage(req, db, logger)

}
