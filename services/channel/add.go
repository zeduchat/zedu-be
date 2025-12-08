package channel

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
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
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/rabbitmq"
	"github.com/hngprojects/telex_be/services/thread"
	"github.com/hngprojects/telex_be/utility"
)

// Reply message fn
func SaveChannelsMsg(req models.CreateMessageRequest, db *storage.Database,
	logger *utility.Logger) (*models.MessageDocument, int, error) {

	var (
		profile       models.Profile
		user          models.User
		channels      models.Channels
		threads       models.ThreadDocument
		agent_message = false
		userChan      models.UserChannels
	)

	userType := "user"

	if req.AgentName != "" && req.UserId == "" {
		agent_message = true
		userType = "bot"
		if req.AgentId == "" {
			return nil, http.StatusUnprocessableEntity, fmt.Errorf("missing agent_id")
		}

		if req.OrgId == "" {
			return nil, http.StatusUnprocessableEntity, fmt.Errorf("missing org_id")
		}

		if err := models.ValidateAgentIDs(db.Postgresql, req.OrgId, []string{req.AgentId}); err != nil {
			return nil, http.StatusNotFound, err
		}

		req.UserId = req.AgentId
	}

	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("invalid thread ID")
	}

	err = threads.GetThreadById(req.ThreadId)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	chanExist := postgresql.CheckExists(db.Postgresql, &channels, "id = ?", req.ChannelsId)
	if !chanExist {
		return nil, http.StatusBadRequest, errors.New("channel does not exist")
	}

	err = profile.GetProfileByUserId(db.Postgresql, req.UserId)

	if err != nil && !agent_message {
		return nil, http.StatusBadRequest, errors.New("failed to get user profile")
	}

	user, err = user.GetUserByID(db.Postgresql, req.UserId)
	if err != nil && !agent_message {
		return nil, http.StatusBadRequest, errors.New("failed to get user")
	}

	messageDoc := models.MessageDocument{
		ID:             utility.GenerateUUID(),
		Content:        req.Content,
		ChannelsID:     req.ChannelsId,
		UserID:         req.UserId,
		ThreadID:       threadId,
		AgentMessage:   agent_message,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		AvatarURL:      profile.AvatarURL,
		Edited:         false,
		UserType:       userType,
		Username:       utility.ThisOrThat(profile.UserName, req.AgentName),
		FullName:       utility.ThisOrThat(profile.FullName, req.AgentName),
		Email:          user.Email,
		Media:          req.Media,
		Mentions:       req.Mentions,
		OrganisationID: channels.OrganisationID,
	}

	updateResp, err := messageDoc.CreateMessage(db, logger)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to save message, error: " + err.Error())
	}

	feed := models.FeedMessageRequest{
		ChannelID: req.ChannelsId,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: messageDoc.UpdatedAt.String(),
		AvatarURL: profile.AvatarURL,
		Type:      "message",
		Content:   req.Content,
		ThreadId:  req.ThreadId,
		Email:     user.Email,
		UserType:  userType,
		UserName:  utility.ThisOrThat(profile.UserName, req.AgentName),
		FullName:  utility.ThisOrThat(profile.FullName, req.AgentName),
		OrgId:     channels.OrganisationID,
		UserId:    req.UserId,
		Media:     req.Media,
		Id:        messageDoc.ID,
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

	dataByte, _ := json.Marshal(feed)

	notifRec := models.PushNotificationRecord{
		ChannelType:  models.Channel,
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

	// increase unread count for channel users
	userChan.ChannelsID = req.ChannelsId
	userChan.UserID = req.UserId
	if req.UserId == "WEBHOOK" {
		userChan.UserID = "00000000-0000-0000-0000-000000000000"
	}
	userChan.OrgId = channels.OrganisationID
	mentionMsg := models.MentionMessage{
		Mention: req.Mentions,
		Content: feed,
	}
	var wg sync.WaitGroup
	mutex := &sync.Mutex{}

	if len(req.Mentions) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			userChan.ProcessMentions(db.Postgresql, req.Mentions, mutex, logger)
		}()
	}

	go func() {
		wg.Wait()
		userChan.SendChannelUnReadUpdate(mutex, logger, models.NewThread, mentionMsg)
	}()

	threadReplyNotif := models.Notification[models.ThreadReply]
	threadReplyNotif.Content = feed
	threadReplyNotif.NotificationId = utility.GenerateUUID()

	err = centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", threads.OrgansationID, threads.UserId), threadReplyNotif)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing thread reply message to user, channelid: %s, with userid: %s error: %v", threads.ChannelsID, threads.UserId, err.Error()))
	}

	logger.Info("Published thread reply message to user : %s", threads.UserId)

	return &messageDoc, http.StatusCreated, nil
}

func EditChannelsMsg(req models.EditMessageRequest, db *gorm.DB, c *gin.Context, logger *utility.Logger) (*models.MessageDocument, int, error) {

	var (
		message   models.Message
		newMsg    models.MessageDocument
		channel   models.Channels
		dmChannel models.DmChannels
		user      models.User
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return &newMsg, http.StatusNotFound, err
	}

	userID, ok := userId.(string)
	if !ok {
		return &newMsg, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	user, err = user.GetUserByID(db, userID)

	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to get user")
	}

	chanExist, _ := channel.CheckChannelExists(db, req.ChannelsId)
	dmChanExist, _ := dmChannel.CheckChannelExists(db, req.ChannelsId, userID)

	if !(dmChanExist || chanExist) {
		return &newMsg, http.StatusNotFound, errors.New("channel does not exist")
	}

	message.ID = req.MessageId

	updateKey := map[string]any{
		"message": req.Content,
		"edited":  true,
	}

	if _, err := message.UpdateMessage(db, updateKey); err != nil {
		return &newMsg, http.StatusNotFound, err
	}

	now := time.Now()
	postgresUpdates := map[string]interface{}{
		"edited":    updateKey["edited"],
		"edited_at": now,
	}
	
	if err := message.UpdateMessageInPostgres(db, postgresUpdates); err != nil {
		logger.Error(fmt.Sprintf("Failed to update message in postgres for message_id: %s, error: %v", req.MessageId, err.Error()))
		return nil, http.StatusInternalServerError, errors.New("failed to update message in postgres: " + err.Error())
	}

	if err := thread.DetectAndAddMentions(message.ID, req.Content, db); err != nil {
		return nil, http.StatusBadRequest, err
	}

	err = newMsg.GetMessageById(db, message.ID)

	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	feed := models.FeedMessageRequest{
		ChannelID: req.ChannelsId,
		CreatedAt: newMsg.CreatedAt.String(),
		UpdatedAt: newMsg.UpdatedAt.String(),
		AvatarURL: user.Profile.AvatarURL,
		Type:      "message",
		Content:   req.Content,
		ThreadId:  req.ThreadId,
		Email:     user.Email,
		UserType:  newMsg.UserType,
		UserName:  user.Profile.UserName,
		FullName:  user.Profile.FullName,
		OrgId:     req.OrgId,
		UserId:    req.UserId,
		Media:     newMsg.Media,
		Id:        req.MessageId,
	}

	notification := models.Notification[models.Updated]
	notification.SectionType = models.ReplySection
	notification.Content = feed
	notification.ModificationDetails = &models.ModificationDetails{
		ThreadId:  req.ThreadId,
		ChannelId: req.ChannelsId,
		MessageId: req.MessageId,
	}

	err = centrifuge.PublishChannel(logger, req.ChannelsId, notification)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing update reply message with destination id: %s error: %v", req.ChannelsId, err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to publish data: " + err.Error())
	}

	return &newMsg, http.StatusOK, nil
}

func DeleteChannelsMsg(req models.EditMessageRequest, db *gorm.DB, logger *utility.Logger) (*models.Message, int, error) {

	var (
		message      models.Message
		newMsg       models.MessageDocument
		channel      models.Channels
		dmChannel    models.DmChannels
		savedMessage models.SavedMessage
	)

	tx := db.Begin()
	if tx.Error != nil {
		logger.Error("Failed to begin transaction: %v", tx.Error)
		return nil, http.StatusInternalServerError, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	chanExist, _ := channel.CheckChannelExists(tx, req.ChannelsId)
	dmChanExist, _ := dmChannel.CheckChannelExists(tx, req.ChannelsId, req.UserId)
	if !(dmChanExist || chanExist) {
		tx.Rollback()
		return nil, http.StatusNotFound, errors.New("channel does not exist")
	}

	if err := newMsg.GetMessageById(tx, req.MessageId); err != nil {
		tx.Rollback()
		return nil, http.StatusBadRequest, errors.New("message not found")
	}

	req.ThreadId = newMsg.ThreadID.String()
	savedMessageIds := models.SavedMessageIds{
		MessageID: newMsg.ID,
		UserID:    req.UserId,
		OrgID:     newMsg.OrganisationID,
		ThreadID:  newMsg.ThreadID.String(),
	}

	if exists := savedMessage.SavedReplyMsgExists(tx, savedMessageIds); exists {
		if err := savedMessage.DeleteSavedMessageByMessageID(tx, savedMessageIds); err != nil {
			tx.Rollback()
			return nil, http.StatusBadRequest, err
		}
	}

	pinnedReply := models.PinnedMessage{
		MessageID:  &newMsg.ID,
		ThreadID:   newMsg.ThreadID.String(),
		ChannelsID: req.ChannelsId,
	}

	if exists := pinnedReply.CheckPinnedReplyExists(tx); exists {
		if err := pinnedReply.DeletePinnedReplyMessageRecord(tx); err != nil {
			tx.Rollback()
			logger.Error("An error occurred while deleting pinned message record: %v", err)
			return nil, http.StatusInternalServerError, err
		}
	}

	updateResp, err := newMsg.DeleteMessage(tx, logger)
	if err != nil {
		tx.Rollback()
		return nil, http.StatusInternalServerError, err
	}

	if _, err := message.DeleteMessageMediaFiles(logger, tx, newMsg.Media); err != nil {
		tx.Rollback()
		return nil, http.StatusInternalServerError, err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, http.StatusInternalServerError, err
	}

	notification := models.Notification[models.Deleted]
	notification.SectionType = models.ReplySection
	notification.UpdateChange = updateResp
	notification.ModificationDetails = &models.ModificationDetails{
		ThreadId:  req.ThreadId,
		ChannelId: req.ChannelsId,
		MessageId: req.MessageId,
	}

	if err = centrifuge.PublishChannel(logger, req.ChannelsId, notification); err != nil {
		logger.Error("Error Publishing to with destination id: %s error: %v", req.ChannelsId, err)
		return nil, http.StatusBadRequest, errors.New("failed to publish data: " + err.Error())
	}

	return nil, http.StatusOK, nil
}

// Reply message fn
func ReplyThreadMessage(req models.CreateMessageRequest, db *storage.Database,
	logger *utility.Logger) (*models.MessageDocument, int, error) {

	var (
		routing_key = "new_message"
		oci         models.OrganisationChannelsIntegrations
		channel     models.Channels
	)

	chanReq := models.ChannelInfo{
		ChannelID: req.ChannelsId,
		UserID:    req.ThreadId, //logicBUG[do not modify]
		// UserID: req.UserId,
	}

	channel_info, err := channel.GetChannelByID(db.Postgresql, chanReq)
	if err != nil {
		logger.Error(fmt.Sprintf("Error checking for organization id: %v", err.Error()))
		return &models.MessageDocument{}, http.StatusBadRequest, fmt.Errorf("failed fetching orgid, error: %v", err)
	}

	req.OrgId = channel_info.OrganisationID

	replyResp, code, err := SaveChannelsMsg(req, db, logger)
	if err != nil {
		return replyResp, code, err
	}

	res, err := oci.CheckHasFilterIntegrations(db.Postgresql, req.ChannelsId)

	if err != nil {
		logger.Error(fmt.Sprintf("Error checking for integration filter status: %v", err.Error()))
		return &models.MessageDocument{}, http.StatusBadRequest, fmt.Errorf("failed fetching filter status, error: %v", err)
	}

	if res {
		returnUrl := fmt.Sprintf("%s/api/v1/channels/backend-queue", config.Config.App.Url)

		feed := models.FeedQueue{
			ChannelsId: req.ChannelsId,
			Content:    req.Content,
			ThreadId:   req.ThreadId,
			ReturnUrl:  returnUrl,
			Type:       "message",
			UserId:     req.UserId,
			OrgId:      req.OrgId,
			Media:      req.Media,
			Mentions:   req.Mentions,
		}

		payload := map[string]any{
			"message_content": map[string]any{
				"channel_id": feed.ChannelsId,
				"message":    feed.Content,
				"thread_id":  feed.ThreadId,
				"type":       feed.Type,
				"user_id":    feed.UserId,
				"org_id":     feed.OrgId,
				"media":      feed.Media,
				"mentions":   feed.Mentions,
			},
			"channel_id": feed.ChannelsId,
			"return_url": feed.ReturnUrl,
		}
		task := "telex_queue_processor.handle_new_message"

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			logger.Error(fmt.Sprintf("Error marshaling payload for integration: %v", err.Error()))
			return &models.MessageDocument{}, http.StatusBadRequest, fmt.Errorf("failed to marshal payload, error: %v", err)
		}

		err = rabbitmq.PushToRabbitQueue(logger, db.Postgresql, string(payloadBytes), routing_key, task)
		if err != nil {
			logger.Error(fmt.Sprintf("Error pushing to RabbitMQ for integration: %v", err.Error()))
			return &models.MessageDocument{}, http.StatusBadRequest, fmt.Errorf("failed to push to RabbitMQ, error: %v", err)
		}
	}

	return replyResp, code, nil
}

func SaveIncomingQueueMsg(req models.FeedQueue, db *storage.Database,
	logger *utility.Logger) error {

	var err error

	reqNew := models.CreateMessageRequest{
		Content:    req.Content,
		ChannelsId: req.ChannelsId,
		ThreadId:   req.ThreadId,
		UserId:     req.UserId,
		OrgId:      req.OrgId,
		AgentName:  req.AgentName,
		Media:      req.Media,
		Mentions:   req.Mentions,
		AgentId:    req.AgentId,
	}

	if req.Type == "message" {

		logger.Info("saving and publishing recieved channel message")
		_, _, err = SaveChannelsMsg(reqNew, db, logger)

	} else if req.Type == "message/thread" {

		reqNew := models.CreateThreadMsgReq{
			Content:    req.Content,
			ChannelsID: req.ChannelsId,
			ThreadId:   req.ThreadId,
			UserId:     req.UserId,
			OrgId:      req.OrgId,
			Media:      req.Media,
			Mentions:   req.Mentions,
			AgentName:  req.AgentName,
			AgentId:    req.AgentId,
		}

		logger.Info("saving and publishing recieved thread message")
		_, err = thread.SaveThreadMessage(reqNew, db, logger)
	}

	if err != nil {
		logger.Error(fmt.Sprintf("Error saving and publishing recieved message: %v", err.Error()))
		return err
	}

	logger.Info("saving and publishing recieved message successfull !!!")
	return err
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

		logger.Info("saving and publishing recieved channel message")
		_, _, err = SaveChannelsMsg(reqNew, db, logger)

	} else if req.Type == "message/thread" {

		reqNew := models.CreateThreadMsgReq{
			Content:    req.Content,
			ChannelsID: req.ChannelsId,
			ThreadId:   req.ThreadId,
			UserId:     req.UserId,
			OrgId:      req.OrgId,
		}

		logger.Info("saving and publishing recieved thread message")
		_, err = thread.SaveThreadMessage(reqNew, db, logger)
	}

	if err != nil {
		logger.Error(fmt.Sprintf("Error saving and publishing recieved message: %v", err.Error()))
		return
	}

	logger.Info("saving and publishing recieved message successfull !!!")
}
