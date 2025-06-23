package dm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/actions"
	push_notifications "github.com/hngprojects/telex_be/services/pushNotifications"
	"github.com/hngprojects/telex_be/services/rabbitmq"
	"github.com/hngprojects/telex_be/services/user"
	"github.com/hngprojects/telex_be/utility"
)

func SaveThreadDmMessage(req models.CreateThreadMsgReq, db *storage.Database, logger *utility.Logger) (*models.ThreadDocument, int, error) {
	var (
		profile models.Profile
		user    models.User
		channel models.DmChannels
	)

	err := profile.GetProfileByUserId(db.Postgresql, req.UserId)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to get user profile")
	}

	user, err = user.GetUserByID(db.Postgresql, req.UserId)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to get user")
	}

	exists, err := channel.CheckChannelExists(db.Postgresql, req.ChannelsID, req.UserId)
	if !exists || err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("channel does not exist: %v", err)
	}

	messageType := "message"
	if req.Type != "" {
		messageType = req.Type
	}

	threadDoc := models.ThreadDocument{
		ID:            utility.GenerateUUID(),
		Username:      profile.UserName,
		Content:       req.Content,
		ChannelsID:    req.ChannelsID,
		Type:          messageType,
		MessageCount:  0,
		AvatarURL:     profile.AvatarURL,
		FullName:      profile.FullName,
		Email:         user.Email,
		CreatedAt:     time.Now().UTC(),
		CurrentStatus: "pending",
		UserId:        req.UserId,
		Messages:      []models.MessageDocument{},
		Status:        "success",
		Edited:        false,
		UserType:      "user",
		Mentions:      req.Mentions,
		Media:         req.Media,
		OrgansationID: channel.OrgId,
	}

	err = threadDoc.CreateThread(db, logger)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to create thread: %v", err)
	}

	username := utility.ThisOrThat(profile.UserName, utility.ThisOrThat(profile.FullName, user.Email))

	feed := models.FeedMessageRequest{
		ChannelID:   req.ChannelsID,
		UserName:    profile.UserName,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		AvatarURL:   profile.AvatarURL,
		Type:        "message",
		Content:     req.Content,
		ThreadId:    threadDoc.ID,
		Email:       user.Email,
		FullName:    profile.FullName,
		UserId:      req.UserId,
		UserType:    "user",
		Media:       req.Media,
		ChannelName: username,
		OrgId:       channel.OrgId,
	}

	err = centrifuge.PublishChannel(logger, req.ChannelsID, feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, error: %v", req.ChannelsID, err))
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to publish webhook data: %v", err)
	}

	dataByte, _ := json.Marshal(feed)

	typeChannelId := req.ChannelsID
	channelType := models.GroupDMChannel

	if channel.ChannelType == "dm" {
		typeChannelId = *channel.ParticipantId
		channelType = models.DMChannel
	}

	notifRec := models.PushNotificationRecord{
		ChannelType: channelType,
		Data:        string(dataByte),
		Sent:        false,
		ChannelId:   typeChannelId,
		Section:     models.ThreadSection,
		Type:        models.NewMessage,
	}

	err = actions.AddPushNotificationToQueue(storage.DB.Redis, notifRec)

	if err != nil {
		logger.Error("Error adding notification to channelid: %s, with orgid: %s error: %v", req.ChannelsID, channel.OrgId, err.Error())
	}

	logger.Info("added notification to queue for channel %s", req.ChannelsID)

	dmChan := models.DmChannels{}
	dmChan.ChannelId = req.ChannelsID
	dmChan.UserId = req.UserId
	dmChan.ChannelType = channel.ChannelType
	dmChan.OrgId = channel.OrgId

	var wg sync.WaitGroup
	mutex := &sync.Mutex{}

	// Add to the wait group for each goroutine that must complete first
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Updating unread thread count")
		dmChan.UpdateUnReadCount(db.Postgresql, mutex, logger)
	}()

	// Run this after the other finish
	go func() {
		wg.Wait()
		dmChan.SendChannelUnReadUpdate(mutex, logger, models.NewThread)
	}()

	return &threadDoc, http.StatusCreated, nil
}

func sendDMMessageToBot(req models.CreateThreadMsgReq, db *storage.Database, logger *utility.Logger) (*models.ThreadDocument, int, error) {
	var (
		profile     models.Profile
		user        models.User
		channel     models.DmChannels
		routing_key = "direct_message"
	)

	err := profile.GetProfileByUserId(db.Postgresql, req.UserId)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to get user profile")
	}

	user, err = user.GetUserByID(db.Postgresql, req.UserId)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to get user")
	}

	exists, err := channel.CheckChannelExists(db.Postgresql, req.ChannelsID, req.UserId)
	if !exists || err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("channel does not exist: %v", err)
	}

	messageType := "message"
	if req.Type != "" {
		messageType = req.Type
	}

	threadDoc := models.ThreadDocument{
		ID:            utility.GenerateUUID(),
		Username:      profile.UserName,
		Content:       req.Content,
		ChannelsID:    req.ChannelsID,
		Type:          messageType,
		MessageCount:  0,
		AvatarURL:     profile.AvatarURL,
		FullName:      profile.FullName,
		Email:         user.Email,
		CreatedAt:     time.Now().UTC(),
		CurrentStatus: "pending",
		UserId:        req.UserId,
		Messages:      []models.MessageDocument{},
		Status:        "success",
		Edited:        false,
		UserType:      "user",
		Mentions:      req.Mentions,
		Media:         req.Media,
		OrgansationID: channel.OrgId,
	}

	err = threadDoc.CreateThread(db, logger)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to create thread: %v", err)
	}

	publishfeed := models.FeedMessageRequest{
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
		UserType:  "user",
		Media:     req.Media,
	}

	err = centrifuge.PublishChannel(logger, publishfeed.ChannelID, publishfeed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to participant id: %s, error: %v", publishfeed.ChannelID, err))
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to publish webhook data: %v", err)
	}

	logger.Info(fmt.Sprintf("Publishing to channel id: %s", publishfeed.ChannelID))

	// Process and perform credit validation rules
	inputputLength := len(req.Content)

	var agentPrice float64 = 0.0 // temp value

	creditUsed := models.CalculateCreditCost(inputputLength, agentPrice)

	// validate credit here
	if !models.OrgHasValidCreditBalance(db.Postgresql, channel.OrgId, creditUsed, logger) {
		logger.Error("Organisation has insufficient credit balance!!")
		return nil, http.StatusBadRequest, fmt.Errorf("organisation has insufficient credit balance")
	}

	returnUrl := fmt.Sprintf("%s/api/v1/dms/bot-dm-response", config.Config.App.Url)
	feed := models.FeedQueue{
		ChannelsId: req.ChannelsID,
		Content:    req.Content,
		ThreadId:   threadDoc.ID,
		ReturnUrl:  returnUrl,
		Type:       "message/thread",
		UserId:     req.UserId,
		OrgId:      req.OrgId,
		Mentions:   req.Mentions,
		Media:      req.Media,
	}

	payload := map[string]interface{}{
		"args": []map[string]interface{}{
			{
				"message_content": map[string]interface{}{
					"channel_id":              feed.ChannelsId,
					"message":                 feed.Content,
					"thread_id":               feed.ThreadId,
					"is_channel_conversation": false,
					"type":                    feed.Type,
					"user_id":                 feed.UserId,
					"org_id":                  feed.OrgId,
					"media":                   feed.Media,
					"mentions":                feed.Mentions,
				},
				"channel_id":  feed.ChannelsId,
				"org_id":      feed.OrgId,
				"return_url":  feed.ReturnUrl,
				"agent_id":    channel.ParticipantId,
				"credit_used": creditUsed,
			},
		},
		"task": "telex_queue_processor.handle_direct_message",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error(fmt.Sprintf("Error marshaling payload for integration: %v", err.Error()))
		return &models.ThreadDocument{}, http.StatusInternalServerError, fmt.Errorf("failed to marshal payload, error: %v", err)
	}

	err = rabbitmq.PushToRabbitQueue(logger, db.Postgresql, string(payloadBytes), routing_key)
	if err != nil {
		logger.Error(fmt.Sprintf("Error pushing to RabbitMQ for integration: %v", err.Error()))
		return &models.ThreadDocument{}, http.StatusInternalServerError, fmt.Errorf("failed to push to RabbitMQ, error: %v", err)
	}

	logger.Info(fmt.Sprintf("Pushed to RabbitMQ for integration: %s", routing_key))

	return &threadDoc, http.StatusCreated, nil
}

// Direct Message thread
func CreateThreadDmMessage(req models.CreateThreadMsgReq, db *storage.Database, logger *utility.Logger) (*models.ThreadDocument, int, error) {

	dmchannel := models.DmChannels{}

	exists := postgresql.CheckExists(db.Postgresql, &dmchannel, "channel_id = ? AND chat_type = ?", req.ChannelsID, "bot")
	req.OrgId = dmchannel.OrgId

	if exists {
		return sendDMMessageToBot(req, db, logger)
	}

	// Create pair room if first message and not a bot
	thread := models.ThreadDocument{
		UserId:     req.UserId,
		ChannelsID: req.ChannelsID,
	}

	pairRoom, code, err := thread.CheckExists()
	if err != nil {
		return &thread, code, err
	}

	if !pairRoom {

		dmchannel := models.DmChannels{}

		res, err := dmchannel.CheckChannelExists(db.Postgresql, req.ChannelsID, req.UserId)

		if !res || err != nil {
			return &thread, http.StatusBadRequest, err
		}

		req.OrgId = dmchannel.OrgId

		if dmchannel.ChatType != "bot" && dmchannel.ChannelType == "dm" {

			pairRoomChan := models.DmChannels{}

			pairRoomChan.ChatType = dmchannel.ChatType
			pairRoomChan.ChannelType = "dm"
			pairRoomChan.UserId = *dmchannel.ParticipantId
			pairRoomChan.ParticipantId = &dmchannel.UserId
			pairRoomChan.ID = utility.GenerateUUID()
			pairRoomChan.ChannelId = dmchannel.ChannelId
			pairRoomChan.OrgId = dmchannel.OrgId

			_, err = pairRoomChan.CreateDmChannel(db.Postgresql)
			if err != nil {
				return &thread, http.StatusInternalServerError, err
			}
		}
	}

	return SaveThreadDmMessage(req, db, logger)
}

func GetAllChannelDmThreads(channelID string, db *gorm.DB, c *gin.Context) ([]models.Threads, *elastic.PaginationResponse, int, error) {
	var (
		accessData         models.Threads
		accessResp         []models.Threads
		paginationResponse *elastic.PaginationResponse
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return nil, nil, http.StatusNotFound, err
	}

	userID, ok := userId.(string)
	if !ok {
		return nil, nil, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	_, code, err := user.GetUser(userID, db)
	if err != nil {
		return nil, nil, code, err
	}

	accessResp, paginationResponse, err = accessData.GetAllThreadsByChannelID(c, db, userID, channelID)

	if err != nil {
		return accessResp, nil, http.StatusInternalServerError, err
	}

	return accessResp, paginationResponse, http.StatusOK, nil
}

func BotResponse(req models.BotReturnRequest, db *storage.Database, logger *utility.Logger, extReq request.ExternalRequest, rds *redis.Client) (*models.ThreadDocument, int, error) {
	var (
		channel  models.DmChannels
		orgAgent models.OrganisationIntegrations
		// user       models.User
	)

	exists, err := channel.CheckChannelExists(db.Postgresql, req.ChannelID, "")
	if !exists || err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("channel does not exist: %v", err)
	}

	exists = postgresql.CheckExists(db.Postgresql, &orgAgent, "integration_id = ?", channel.ParticipantId)
	if !exists {
		return nil, http.StatusBadRequest, fmt.Errorf("agent does not exist")
	}

	threadDoc := models.ThreadDocument{
		ID:            utility.GenerateUUID(),
		Username:      orgAgent.AppName,
		Content:       req.Content,
		ChannelsID:    req.ChannelID,
		Type:          "message",
		MessageCount:  0,
		AvatarURL:     orgAgent.AppLogo,
		FullName:      orgAgent.AppName,
		Email:         "agent",
		CreatedAt:     time.Now().UTC(),
		CurrentStatus: "pending",
		UserId:        *channel.ParticipantId,
		Messages:      []models.MessageDocument{},
		Status:        "success",
		UserType:      "bot",
		Edited:        false,
		Mentions:      req.Mentions,
		Media:         req.Media,
		OrgansationID: channel.OrgId,
		State:         req.State,
	}

	err = threadDoc.CreateThread(db, logger)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to create thread: %v", err)
	}

	feed := models.FeedMessageRequest{
		ChannelID: req.ChannelID,
		UserName:  orgAgent.AppName,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		AvatarURL: orgAgent.AppLogo,
		Type:      "message",
		Content:   req.Content,
		ThreadId:  threadDoc.ID,
		Email:     "agent",
		FullName:  orgAgent.AppName,
		UserId:    *channel.ParticipantId,
		Media:     req.Media,
		UserType:  "bot",
		State:     req.State,
	}

	err = centrifuge.PublishChannel(logger, req.ChannelID, feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, error: %v", req.ChannelID, err))
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to publish webhook data: %v", err)
	}
	logger.Info(fmt.Sprintf("Publishing to channel id: %s", req.ChannelID))

	pushReq := models.PushFCMRequest{
		ChannelName: feed.UserName,
		UserId:      *channel.ParticipantId,
		Message:     req.Content,
		TimeStamp:   threadDoc.CreatedAt.String(),
		AvatarUrl:   feed.AvatarURL,
	}

	err = push_notifications.PushFCMToUser(pushReq, logger, db.Postgresql)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to send push notification to user %s: %v", *channel.ParticipantId, err))
	}

	var creditUsed float64 = req.CreditUsed
	if req.OperationPrice != nil && *req.OperationPrice > 0 {
		creditUsed = req.CreditUsed * (*req.OperationPrice / 100) // apply operation price
	}

	credit_usage := models.CreditUsage{
		ID:             utility.GenerateUUID(),
		OrganisationID: channel.OrgId,
		Amount:         creditUsed,
		AgentID:        *channel.ParticipantId,
		UserID:         nil,
	}

	err = credit_usage.UpdateOrCreateDailyCredit(db.Postgresql, creditUsed)
	if err != nil {
		logger.Error("failed to create/update credit usage!!")
		return nil, http.StatusBadRequest, fmt.Errorf("failed to create/update organisation credit usage: %v", err)
	}

	// create agent bill to an organization only if special operation was performed
	if req.OperationPrice != nil && *req.OperationPrice > 0 {
		err = orgAgent.CreateOrUpdateBillFromUsage(db.Postgresql, &credit_usage)
		if err != nil {
			logger.Error("failed to create/update agent bill")
			return nil, http.StatusBadRequest, fmt.Errorf("failed to create/update agent bill: %v", err)
		}
	}

	// update organization credit balance
	if err = models.UpdateOrgCreditBalance(db.Postgresql, channel.OrgId); err != nil {
		logger.Error("Organisation credit Recalculation failed")
		return nil, http.StatusBadRequest, fmt.Errorf("organisation credit recalculation failed: %v", err)
	}

	return &threadDoc, http.StatusCreated, nil
}
