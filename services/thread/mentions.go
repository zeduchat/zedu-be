package thread

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/typesense/typesense-go/v2/typesense"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	tydb "github.com/hngprojects/telex_be/pkg/repository/storage/typesense"
	"github.com/hngprojects/telex_be/services/rabbitmq"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/channels_utility"
)

func SaveThreadMessage(req models.CreateThreadMsgReq, db *gorm.DB, typesenseDb *typesense.Client, logger *utility.Logger) (*models.Threads, error) {

	var (
		profile models.Profile
		user    models.User
	)

	err := profile.GetProfileByUserId(db, req.UserId)

	if err != nil {
		return nil, errors.New("failed to get user profile")
	}

	user, err = user.GetUserByID(db, req.UserId)

	if err != nil {
		return nil, errors.New("failed to get user")
	}

	thread := models.Threads{
		ID:            req.ThreadId,
		Username:      profile.UserName,
		Content:       req.Content,
		ChannelsID:    req.ChannelsID,
		Type:          "message",
		MessageCount:  0,
		AvatarURL:     profile.AvatarURL,
		FullName:      profile.FullName,
		Email:         user.Email,
		CurrentStatus: "pending",
	}

	if err = thread.CreateThread(db, typesenseDb); err != nil {
		return nil, err
	}

	feed := models.FeedMessageRequest{
		ChannelID: req.ChannelsID,
		UserName:  profile.UserName,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		AvatarURL: profile.AvatarURL,
		Type:      "message",
		Content:   req.Content,
		ThreadId:  req.ThreadId,
		Email:     user.Email,
		FullName:  profile.FullName,
	}

	err = centrifuge.BroadcastChannel(logger, req.ChannelsID, feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Broadcasting to channelid: %s, error: %v", req.ChannelsID, err.Error()))
		return nil, errors.New("failed to broadcast webhook data: " + err.Error())
	}

	return &thread, nil
}

func CreateThreadMessage(req models.CreateThreadMsgReq, db *gorm.DB, typesenseDb *typesense.Client, logger *utility.Logger) (*models.Threads, error) {

	var (
		routing_key = "new_message"
		oci         models.OrganisationChannelsIntegrations
	)

	res, err := oci.CheckHasFilterIntegrations(db, req.ChannelsID)

	if err != nil {
		logger.Error(fmt.Sprintf("Error checking for integration filter status: %v", err.Error()))
		return &models.Threads{}, fmt.Errorf("failed fetching filter status, error: %v", err)
	}

	if !res {
		return SaveThreadMessage(req, db, typesenseDb, logger)
	}

	returnUrl := fmt.Sprintf("%s/api/v1/channels/backend-queue", config.Config.App.Url)

	feed := models.FeedQueue{
		ChannelsId: req.ChannelsID,
		Content:    req.Content,
		ThreadId:   req.ThreadId,
		ReturnUrl:  returnUrl,
		Type:       "message",
		UserId:     req.UserId,
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
		return &models.Threads{}, fmt.Errorf("failed to marshal payload, error: %v", err)
	}

	err = rabbitmq.PushToRabbitQueue(logger, db, string(payloadBytes), routing_key)
	if err != nil {
		logger.Error(fmt.Sprintf("Error pushing to RabbitMQ for integration: %v", err.Error()))
		return &models.Threads{}, fmt.Errorf("failed to push to RabbitMQ, error: %v", err)
	}

	return &models.Threads{}, nil

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
