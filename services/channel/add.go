package channel

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	push_notifications "github.com/hngprojects/telex_be/services/pushNotifications"
	"github.com/hngprojects/telex_be/services/rabbitmq"
	"github.com/hngprojects/telex_be/services/thread"
	"github.com/hngprojects/telex_be/services/user"
	"github.com/hngprojects/telex_be/utility"
)

// Reply message fn
func SaveChannelsMsg(req models.CreateMessageRequest, db *storage.Database,
	logger *utility.Logger) (*models.MessageDocument, int, error) {

	var (
		profile  models.Profile
		user     models.User
		channels models.Channels
	)

	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("invalid thread ID")
	}

	chanExist := postgresql.CheckExists(db.Postgresql, &channels, "id = ?", req.ChannelsId)

	if !chanExist {
		return nil, http.StatusBadRequest, errors.New("channel does not exist")
	}

	err = profile.GetProfileByUserId(db.Postgresql, req.UserId)

	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to get user profile")
	}

	user, err = user.GetUserByID(db.Postgresql, req.UserId)

	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to get user")
	}

	messageDoc := models.MessageDocument{
		ID:         utility.GenerateUUID(),
		Content:    req.Content,
		ChannelsID: req.ChannelsId,
		UserID:     req.UserId,
		ThreadID:   threadId,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
		AvatarURL:  profile.AvatarURL,
		Edited:     false,
		Username:   profile.UserName,
		FullName:   profile.FullName,
		Email:      user.Email,
	}

	err = messageDoc.CreateMessage(db, logger)

	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to save message, error: " + err.Error())
	}

	if err := thread.DetectAndAddMentions(messageDoc.ID, req.Content, db.Postgresql); err != nil {
		return &messageDoc, http.StatusBadRequest, err
	}

	feed := models.FeedMessageRequest{
		ChannelID: req.ChannelsId,
		UserName:  profile.UserName,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		AvatarURL: profile.AvatarURL,
		Type:      "message",
		Content:   req.Content,
		ThreadId:  req.ThreadId,
		Email:     user.Email,
		FullName:  profile.FullName,
		OrgId:     req.OrgId,
		UserId:    req.UserId,
	}

	err = centrifuge.BroadcastChannel(logger, threadId.String(), feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Broadcasting to threadId: %s, error: %v", threadId.String(), err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to broadcast webhook data: " + err.Error())
	}

	notification := models.Notifcation[models.NewMessage]
	notification.SectionType = models.ReplySection
	notification.Content = feed

	err = centrifuge.BroadcastChannel(logger, req.OrgId, notification)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Broadcasting to with orgid: %s error: %v", req.OrgId, err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to broadcast webhook data: " + err.Error())
	}

	pushReq := models.PushFCMRequest{
		ChannelId:   req.ChannelsId,
		ChannelName: channels.Name,
		UserId:      req.UserId,
		Message:     req.Content,
		TimeStamp: messageDoc.CreatedAt.String(),
	}

	err = push_notifications.PushFCMToUsers(pushReq, logger, db.Postgresql)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to send push notifcation to channel users")
	}

	return &messageDoc, http.StatusCreated, nil
}

func EditChannelsMsg(req models.EditMessageRequest, db *gorm.DB, c *gin.Context, logger *utility.Logger) (*models.MessageDocument, int, error) {

	var (
		message      models.Message
		newMsg       models.MessageDocument
		channel      models.Channels
		dmChannel    models.DmChannels
		broadcastDst string
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return &newMsg, http.StatusNotFound, err
	}

	userID, ok := userId.(string)
	if !ok {
		return &newMsg, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	_, code, err := user.GetUser(userID, db)
	if err != nil {
		return &newMsg, code, err
	}

	chanExist, _ := channel.CheckChannelExists(db, req.ChannelsId)
	dmChanExist, _ := dmChannel.CheckChannelExists(db, req.ChannelsId)

	if !(dmChanExist || chanExist) {
		return &newMsg, http.StatusNotFound, errors.New("channel does not exist")
	}

	message.ID = req.MessageId

	updateKey := map[string]interface{}{
		"message": req.Content,
		"edited":  true,
	}

	if _, err := message.UpdateMessage(db, updateKey); err != nil {
		return &newMsg, http.StatusNotFound, err
	}

	if err := thread.DetectAndAddMentions(message.ID, req.Content, db); err != nil {
		return nil, http.StatusBadRequest, err
	}

	if channel.OrganisationID != "" {
		broadcastDst = channel.OrganisationID

	} else {
		broadcastDst = *dmChannel.ParticipantId
	}

	notification := models.Notifcation[models.Updated]
	notification.SectionType = models.ReplySection
	notification.ModifcationDetails = models.ModifcationDetails{
		ThreadId:  req.ThreadId,
		ChannelId: req.ChannelsId,
		MessageId: req.MessageId,
	}

	err = centrifuge.BroadcastChannel(logger, broadcastDst, notification)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Broadcasting to with destination id: %s error: %v", broadcastDst, err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to broadcast data: " + err.Error())
	}

	err = newMsg.GetMessageById(db, message.ID)

	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return &newMsg, http.StatusOK, nil
}

func DeleteChannelsMsg(req models.EditMessageRequest, db *gorm.DB, logger *utility.Logger) (*models.Message, int, error) {

	var (
		message      models.Message
		newMsg       models.MessageDocument
		channel      models.Channels
		dmChannel    models.DmChannels
		broadcastDst string
		chanParts    []models.ChannelParticipant
	)

	message.ID = req.MessageId

	chanExist, _ := channel.CheckChannelExists(db, req.ChannelsId)
	dmChanExist, _ := dmChannel.CheckChannelExists(db, req.ChannelsId)

	if !(dmChanExist || chanExist) {
		return nil, http.StatusNotFound, errors.New("channel does not exist")
	}

	err := newMsg.GetMessageById(db, message.ID)

	if err != nil {
		return nil, http.StatusBadRequest, errors.New("message not found")
	}

	req.ThreadId = newMsg.ThreadID.String()

	if _, err := message.DeleteMessage(); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	notification := models.Notifcation[models.Deleted]
	notification.SectionType = models.ReplySection
	notification.ModifcationDetails = models.ModifcationDetails{
		ThreadId:  req.ThreadId,
		ChannelId: req.ChannelsId,
		MessageId: req.MessageId,
	}

	if channel.OrganisationID != "" {
		broadcastDst = channel.OrganisationID
	} else {
		if dmChannel.ChannelType == "dm" {
			broadcastDst = *dmChannel.ParticipantId
		}
	}

	if dmChannel.ChannelType == "group_dm" && channel.OrganisationID == "" {
		err := postgresql.SelectAllFromDb(db, "", &chanParts, "channel_id = ?", dmChannel.ChannelId)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch channel participants: %s", err)
		}

		for _, participant := range chanParts {
			if participant.UserId != req.UserId {
				err = centrifuge.BroadcastChannel(logger, participant.UserId, notification)
				if err != nil {
					logger.Error(fmt.Sprintf("Error Broadcasting to with destination id: %s error: %v", broadcastDst, err.Error()))
				}
			}
		}

		return nil, http.StatusOK, nil
	}

	err = centrifuge.BroadcastChannel(logger, broadcastDst, notification)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Broadcasting to with destination id: %s error: %v", broadcastDst, err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to broadcast data: " + err.Error())
	}

	return nil, http.StatusOK, nil
}

// Reply message fn
func AddChannelsMsg(req models.CreateMessageRequest, db *storage.Database,
	logger *utility.Logger) (*models.MessageDocument, int, error) {

	var (
		routing_key = "new_message"
		oci         models.OrganisationChannelsIntegrations
		channel     models.Channels
	)

	res, err := oci.CheckHasFilterIntegrations(db.Postgresql, req.ChannelsId)

	if err != nil {
		logger.Error(fmt.Sprintf("Error checking for integration filter status: %v", err.Error()))
		return &models.MessageDocument{}, http.StatusBadRequest, fmt.Errorf("failed fetching filter status, error: %v", err)
	}

	chanReq := models.ChannelInfo{
		ChannelID: req.ChannelsId,
		UserID:    req.ThreadId,
	}

	channel_info, err := channel.GetChannelsByID(db.Postgresql, chanReq)

	if err != nil {
		logger.Error(fmt.Sprintf("Error checking for organization id: %v", err.Error()))
		return &models.MessageDocument{}, http.StatusBadRequest, fmt.Errorf("failed fetching orgid, error: %v", err)
	}

	req.OrgId = channel_info.OrganisationID

	if !res {
		return SaveChannelsMsg(req, db, logger)
	}

	returnUrl := fmt.Sprintf("%s/api/v1/channels/backend-queue", config.Config.App.Url)

	feed := models.FeedQueue{
		ChannelsId: req.ChannelsId,
		Content:    req.Content,
		ThreadId:   req.ThreadId,
		ReturnUrl:  returnUrl,
		Type:       "message",
		UserId:     req.UserId,
		OrgId:      req.OrgId,
	}

	payload := map[string]interface{}{
		"args": []map[string]interface{}{
			{
				"message_content": map[string]interface{}{
					"channel_id": feed.ChannelsId,
					"message":    feed.Content,
					"thread_id":  feed.ThreadId,
					"type":       feed.Type,
					"user_id":    feed.UserId,
					"org_id":     feed.OrgId,
				},
				"channel_id": feed.ChannelsId,
				"return_url": feed.ReturnUrl,
			},
		},
		"task": "telex_queue_processor.handle_new_message",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error(fmt.Sprintf("Error marshaling payload for integration: %v", err.Error()))
		return &models.MessageDocument{}, http.StatusBadRequest, fmt.Errorf("failed to marshal payload, error: %v", err)
	}

	err = rabbitmq.PushToRabbitQueue(logger, db.Postgresql, string(payloadBytes), routing_key)
	if err != nil {
		logger.Error(fmt.Sprintf("Error pushing to RabbitMQ for integration: %v", err.Error()))
		return &models.MessageDocument{}, http.StatusBadRequest, fmt.Errorf("failed to push to RabbitMQ, error: %v", err)
	}

	return &models.MessageDocument{}, http.StatusOK, nil
}

func SaveIncomingQueueMsg(req models.FeedQueue, db *storage.Database,
	logger *utility.Logger) {

	var err error

	reqNew := models.CreateMessageRequest{
		Content:    req.Content,
		ChannelsId: req.ChannelsId,
		ThreadId:   req.ThreadId,
		UserId:     req.UserId,
		OrgId:      req.OrgId,
	}

	if req.Type == "message" {

		logger.Info("saving and broadcasting recieved channel message")
		_, _, err = SaveChannelsMsg(reqNew, db, logger)

	} else if req.Type == "message/thread" {

		reqNew := models.CreateThreadMsgReq{
			Content:    req.Content,
			ChannelsID: req.ChannelsId,
			ThreadId:   req.ThreadId,
			UserId:     req.UserId,
			OrgId:      req.OrgId,
		}

		logger.Info("saving and broadcasting recieved thread message")
		_, err = thread.SaveThreadMessage(reqNew, db, logger)
	}

	if err != nil {
		logger.Error(fmt.Sprintf("Error saving and broadcasting recieved message: %v", err.Error()))
		return
	}

	logger.Info("saving and broadcasting recieved message successfull !!!")
}

func UpdateIncomingQueueMsg(req models.FeedQueue, db *storage.Database,
	logger *utility.Logger) {

	var err error

	reqNew := models.CreateMessageRequest{
		Content:    req.Content,
		ChannelsId: req.ChannelsId,
		ThreadId:   req.ThreadId,
		UserId:     req.UserId,
		OrgId:      req.OrgId,
	}

	if req.Type == "message" {

		logger.Info("saving and broadcasting recieved channel message")
		_, _, err = SaveChannelsMsg(reqNew, db, logger)

	} else if req.Type == "message/thread" {

		reqNew := models.CreateThreadMsgReq{
			Content:    req.Content,
			ChannelsID: req.ChannelsId,
			ThreadId:   req.ThreadId,
			UserId:     req.UserId,
			OrgId:      req.OrgId,
		}

		logger.Info("saving and broadcasting recieved thread message")
		_, err = thread.SaveThreadMessage(reqNew, db, logger)
	}

	if err != nil {
		logger.Error(fmt.Sprintf("Error saving and broadcasting recieved message: %v", err.Error()))
		return
	}

	logger.Info("saving and broadcasting recieved message successfull !!!")
}
