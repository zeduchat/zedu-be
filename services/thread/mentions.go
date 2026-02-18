package thread

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/typesense/typesense-go/v2/typesense"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tydb "github.com/hngprojects/telex_be/pkg/repository/storage/typesense"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/rabbitmq"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/channels_utility"
)

func SaveThreadMessage(req models.CreateThreadMsgReq, db *storage.Database, logger *utility.Logger) (*models.ThreadDocument, error) {

	var (
		profile       models.Profile
		user          models.User
		channel       models.Channels
		userChan      models.UserChannels
		agent_message = false
		channelType   string
	)

	userType := "user"

	if req.AgentName != "" && req.UserId == "" {
		agent_message = true
		userType = "bot"
		if req.AgentId == "" {
			return nil, fmt.Errorf("missing agent_id")
		}

		if req.OrgId == "" && req.AgentId != "WEBHOOK" {
			return nil, fmt.Errorf("missing org_id")
		}

		if req.AgentId != "WEBHOOK" {
			if err := models.ValidateAgentIDs(db.Postgresql, req.OrgId, []string{req.AgentId}); err != nil {
				return nil, err
			}
		}
		req.UserId = req.AgentId
	}

	err := profile.GetProfileByUserId(db.Postgresql, req.UserId)

	if err != nil && !agent_message {
		return nil, fmt.Errorf("failed to get profile: invalid webhook payload")
	}

	user, err = user.GetUserByID(db.Postgresql, req.UserId)

	if err != nil && !agent_message {
		return nil, fmt.Errorf("failed to get user: invalid webhook payload")
	}

	ch, err := channel.CheckChannelExists(db.Postgresql, req.ChannelsID)

	if !ch || err != nil {
		return nil, fmt.Errorf("channel does not exist: invalid channel Id")
	}

	if channel.IsPrivate {
		channelType = "private"
	} else {
		channelType = "public"
	}

	messageType := "message"
	if req.Type != "" {
		messageType = req.Type
	}

	threadDoc := models.ThreadDocument{
		ID:             req.ThreadId,
		Username:       utility.ThisOrThat(profile.UserName, req.AgentName),
		Content:        req.Content,
		ChannelsID:     req.ChannelsID,
		Type:           messageType,
		MessageCount:   0,
		AvatarURL:      profile.AvatarURL,
		FullName:       utility.ThisOrThat(profile.FullName, req.AgentName),
		Email:          user.Email,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		CurrentStatus:  "pending",
		UserType:       userType,
		UserId:         req.UserId,
		Messages:       []models.MessageDocument{},
		ChannelName:    channel.Name,
		ChannelType:    channelType,
		Status:         "success",
		Edited:         false,
		Mentions:       req.Mentions,
		Media:          req.Media,
		OrganisationID: channel.OrganisationID,
	}
	err = threadDoc.CreateThread(db, logger)
	if err != nil {
		return nil, err
	}

	if len(req.Media) > 0 {
		fileIDs := make([]string, len(req.Media))
		for i, file := range req.Media {
			fileIDs[i] = file.ID
		}

		updateAttachedFilesMetadata(db, logger, fileIDs, req.ChannelsID, threadDoc.ID)
	}

	feed := models.FeedMessageRequest{
		ChannelID:   req.ChannelsID,
		UserName:    utility.ThisOrThat(profile.UserName, req.AgentName),
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		AvatarURL:   profile.AvatarURL,
		Type:        "message",
		Content:     req.Content,
		ThreadId:    threadDoc.ID,
		Email:       user.Email,
		UserType:    userType,
		FullName:    utility.ThisOrThat(profile.FullName, req.AgentName),
		UserId:      req.UserId,
		OrgId:       channel.OrganisationID,
		Media:       req.Media,
		ChannelName: channel.Name,
		ChannelType: channelType,
	}

	err = centrifuge.PublishChannel(logger, req.ChannelsID, feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, error: %v", req.ChannelsID, err.Error()))
		return nil, fmt.Errorf("failed to publish thread data")
	}

	dataByte, _ := json.Marshal(feed)

	notifRec := models.PushNotificationRecord{
		ChannelType: models.Channel,
		Data:        string(dataByte),
		Sent:        false,
		ChannelId:   req.ChannelsID,
		Section:     models.ThreadSection,
		Type:        models.NewMessage,
	}

	err = actions.AddPushNotificationToQueue(storage.DB.Redis, notifRec)

	if err != nil {
		logger.Error("Error adding notification to channelid: %s, with orgid: %s error: %v", req.ChannelsID, req.OrgId, err.Error())
	}

	logger.Info("added notification to queue for channel %s", req.ChannelsID)

	// increase unread count for channel users
	userChan.ChannelsID = req.ChannelsID
	userChan.UserID = req.UserId
	if req.UserId == "WEBHOOK" {
		userChan.UserID = "00000000-0000-0000-0000-000000000000"
	}
	userChan.OrgId = channel.OrganisationID
	mentionMsg := models.MentionMessage{
		Mention: req.Mentions,
		Content: feed,
	}
	var wg sync.WaitGroup
	mutex := &sync.Mutex{}

	// Add to the wait group for each goroutine that must complete first
	wg.Add(1)
	go func() {
		defer wg.Done()
		userChan.UpdateUnReadCount(db.Postgresql, mutex, logger)
	}()

	if len(req.Mentions) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			userChan.ProcessMentions(db.Postgresql, req.Mentions, mutex, logger)
		}()
	}

	// Run this after the others finish
	go func() {
		wg.Wait()
		userChan.SendChannelUnReadUpdate(mutex, logger, models.NewThread, mentionMsg)
	}()

	return &threadDoc, nil
}

// main channel thread
func CreateThreadMessage(req models.CreateThreadMsgReq, db *storage.Database, logger *utility.Logger) (*models.ThreadDocument, error) {

	var (
		routing_key = "new_message"
		oci         models.OrganisationChannelsIntegrations
		channel     models.Channels
	)

	chanReq := models.ChannelInfo{
		ChannelID: req.ChannelsID,
		UserID:    req.UserId,
	}

	channel_info, err := channel.GetChannelByID(db, chanReq)

	if err != nil {
		logger.Error(fmt.Sprintf("Error checking for organization id: %v", err.Error()))
		return &models.ThreadDocument{}, fmt.Errorf("failed fetching orgid, error: %v", err)
	}

	req.OrgId = channel_info.OrganisationID
	req.ThreadId = utility.GenerateUUID()

	threadResp, err := SaveThreadMessage(req, db, logger)
	if err != nil {
		return threadResp, err
	}

	res, err := oci.CheckHasFilterIntegrations(db.Postgresql, req.ChannelsID)
	if err != nil {
		logger.Error("Error checking for agents for channel: %v", err.Error())
		return &models.ThreadDocument{}, fmt.Errorf("failed fetching agents, error: %v", err)
	}

	if res {
		returnUrl := fmt.Sprintf("%s/api/v1/channels/backend-queue", config.Config.App.Url)

		feed := models.FeedQueue{
			ChannelsId: req.ChannelsID,
			Content:    req.Content,
			ThreadId:   req.ThreadId,
			ReturnUrl:  returnUrl,
			Type:       "message/thread",
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
			return &models.ThreadDocument{}, fmt.Errorf("failed to marshal payload, error: %v", err)
		}

		err = rabbitmq.PushToRabbitQueue(logger, db.Postgresql, string(payloadBytes), routing_key, task)
		if err != nil {
			logger.Error(fmt.Sprintf("Error pushing to RabbitMQ for integration: %v", err.Error()))
			return &models.ThreadDocument{}, fmt.Errorf("failed to push to RabbitMQ, error: %v", err)
		}
	}

	return threadResp, nil
}

func DetectAndAddMentions(messageID string, content string, db *gorm.DB) error {
	mentions := channels_utility.DetectMentions(content)

	for _, username := range mentions {
		var user models.Profile
		user, err := user.GetUserByUsername(db, username)
		if err != nil {
			continue
		}

		mention := models.Mentions{
			ID:        utility.GenerateUUID(),
			MessageID: messageID,
			UserID:    user.ID,
		}

		if err := mention.CreateMention(db); err != nil {
			return err
		}
	}

	return nil
}


func updateAttachedFilesMetadata(db *storage.Database, logger *utility.Logger, fileIDs []string, channelID, messageID string) {
	if len(fileIDs) == 0 {
		return
	}

	updates := map[string]interface{}{}
	if channelID != "" {
		updates["channel_id"] = channelID
	}
	if messageID != "" {
		updates["message_id"] = messageID
	}

	if len(updates) == 0 {
		return
	}

	err := db.Postgresql.Model(&models.File{}).
		Where("id IN ?", fileIDs).
		Updates(updates).Error

	if err != nil {
		logger.Error("Failed to update file metadata",
			"file_ids", fileIDs,
			"channel_id", channelID,
			"message_id", messageID,
			"error", err)
	}
}

func SearchChannel(channelID, searchWords string, db *gorm.DB, c *gin.Context, typesenseDb *typesense.Client) (*[]map[string]any, int, error) {
	searchField := "username,content,event_name,action_type"
	documents, err := tydb.SearchDocuments(typesenseDb, channelID, searchWords, searchField)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("error searching documents: %v", err)
	}

	return &documents, http.StatusOK, nil
}
