package thread

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/typesense/typesense-go/v2/typesense"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tydb "github.com/hngprojects/telex_be/pkg/repository/storage/typesense"
	push_notifications "github.com/hngprojects/telex_be/services/pushNotifications"
	"github.com/hngprojects/telex_be/services/rabbitmq"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/channels_utility"
)

func SaveThreadMessage(req models.CreateThreadMsgReq, db *storage.Database, logger *utility.Logger) (*models.ThreadDocument, error) {

	var (
		profile models.Profile
		user    models.User
		channel models.Channels
	)

	err := profile.GetProfileByUserId(db.Postgresql, req.UserId)

	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %v", err)
	}

	user, err = user.GetUserByID(db.Postgresql, req.UserId)

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	ch, err := channel.CheckChannelExists(db.Postgresql, req.ChannelsID)

	if !ch || err != nil {
		return nil, fmt.Errorf("channel does not exist: %v", err)
	}

	threadDoc := models.ThreadDocument{
		ID:            req.ThreadId,
		Username:      profile.UserName,
		Content:       req.Content,
		ChannelsID:    req.ChannelsID,
		Type:          "message",
		MessageCount:  0,
		AvatarURL:     profile.AvatarURL,
		FullName:      profile.FullName,
		Email:         user.Email,
		CreatedAt:     time.Now().UTC(),
		CurrentStatus: "pending",
		UserId:        req.UserId,
		Messages:      []models.MessageDocument{},
		ChannelName:   channel.Name,
		Status:        "success",
		Edited:        false,
		Mentions:      req.Mentions,
		Media:         req.Media,
	}
	err = threadDoc.CreateThread(db, logger)
	if err != nil {
		return nil, err
	}

	feed := models.FeedMessageRequest{
		ChannelID: req.ChannelsID,
		UserName:  profile.UserName,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		AvatarURL: profile.AvatarURL,
		Type:      "message",
		Content:   req.Content,
		ThreadId:  threadDoc.ID,
		Email:     user.Email,
		FullName:  profile.FullName,
		UserId:    req.UserId,
		OrgId:     req.OrgId,
	}

	err = centrifuge.PublishChannel(logger, req.ChannelsID, feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, error: %v", req.ChannelsID, err.Error()))
		return nil, fmt.Errorf("failed to publish thread data")
	}

	notification := models.Notification[models.NewMessage]
	notification.SectionType = models.ThreadSection
	notification.Content = feed

	err = centrifuge.PublishChannel(logger, req.OrgId, notification)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, with orgid: %s error: %v", req.ChannelsID, req.OrgId, err.Error()))
		return nil, fmt.Errorf("failed to publish thread data")
	}

	// Push notification to channel users

	pushReq := models.PushFCMRequest{
		ChannelId:   req.ChannelsID,
		ChannelName: channel.Name,
		UserId:      req.UserId,
		Message:     req.Content,
	}

	err = push_notifications.PushFCMToUsers(pushReq, logger, db.Postgresql)
	if err != nil {
		return nil, fmt.Errorf("failed to send push notifcation to channel users")
	}

	logger.Info("sent push notification to channel users")

	return &threadDoc, nil
}

// main channel thread
func CreateThreadMessage(req models.CreateThreadMsgReq, db *storage.Database, logger *utility.Logger) (*models.ThreadDocument, error) {

	var (
		routing_key = "new_message"
		oci         models.OrganisationChannelsIntegrations
		channel     models.Channels
	)

	res, err := oci.CheckHasFilterIntegrations(db.Postgresql, req.ChannelsID)
	if err != nil {
		logger.Error(fmt.Sprintf("Error checking for integration filter status: %v", err.Error()))
		return &models.ThreadDocument{}, fmt.Errorf("failed fetching filter status, error: %v", err)
	}

	chanReq := models.ChannelInfo{
		ChannelID: req.ChannelsID,
		UserID:    req.UserId,
	}

	channel_info, err := channel.GetChannelsByID(db.Postgresql, chanReq)

	if err != nil {
		logger.Error(fmt.Sprintf("Error checking for organization id: %v", err.Error()))
		return &models.ThreadDocument{}, fmt.Errorf("failed fetching orgid, error: %v", err)
	}

	req.OrgId = channel_info.OrganisationID

	if !res {
		return SaveThreadMessage(req, db, logger)
	}

	returnUrl := fmt.Sprintf("%s/api/v1/channels/backend-queue", config.Config.App.Url)

	feed := models.FeedQueue{
		ChannelsId: req.ChannelsID,
		Content:    req.Content,
		ThreadId:   req.ThreadId,
		ReturnUrl:  returnUrl,
		Type:       "message/thread",
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
					// "is_channel_conversation": true,
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
		return &models.ThreadDocument{}, fmt.Errorf("failed to marshal payload, error: %v", err)
	}

	err = rabbitmq.PushToRabbitQueue(logger, db.Postgresql, string(payloadBytes), routing_key)
	if err != nil {
		logger.Error(fmt.Sprintf("Error pushing to RabbitMQ for integration: %v", err.Error()))
		return &models.ThreadDocument{}, fmt.Errorf("failed to push to RabbitMQ, error: %v", err)
	}

	return &models.ThreadDocument{}, nil
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

func SearchChannel(channelID, searchWords string, db *gorm.DB, c *gin.Context, typesenseDb *typesense.Client) (*[]map[string]interface{}, int, error) {
	searchField := "username,content,event_name,action_type"
	documents, err := tydb.SearchDocuments(typesenseDb, channelID, searchWords, searchField)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("error searching documents: %v", err)
	}

	return &documents, http.StatusOK, nil
}
