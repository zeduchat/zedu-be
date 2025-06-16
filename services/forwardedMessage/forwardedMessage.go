package forwardedMessage

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/utility"

	"sync"
)

func ForwardReplyMessage(db *storage.Database, req models.ForwardReplyMessageRequest, logger *utility.Logger, userID string) (*models.ThreadDocument, error) {
	var (
		originalMsg models.MessageDocument
		user        models.User
		profile     models.Profile
		channels    models.Channels
		userChan    models.UserChannels
	)

	chanExist := postgresql.CheckExists(db.Postgresql, &channels, "id = ?", req.ChannelsId)
	if !chanExist {
		return nil, errors.New("channel does not exist")
	}

	if err := profile.GetProfileByUserId(db.Postgresql, userID); err != nil {
		return nil, errors.New("failed to get user profile")
	}

	user, userErr := user.GetUserByID(db.Postgresql, userID)
	if userErr != nil {
		return nil, errors.New("failed to get user")
	}

	if err := originalMsg.GetMessageById(db.Postgresql, req.MessageId); err != nil {
		return nil, err
	}

	threadDoc := models.ThreadDocument{
		ID:              utility.GenerateUUID(),
		Username:        profile.UserName,
		Content:         originalMsg.Content,
		ChannelsID:      req.ChannelsId,
		Type:            "message",
		MessageCount:    0,
		AvatarURL:       profile.AvatarURL,
		FullName:        profile.FullName,
		Email:           user.Email,
		CreatedAt:       time.Now().UTC(),
		CurrentStatus:   "pending",
		UserType:        originalMsg.UserType,
		UserId:          userID,
		ChannelName:     channels.Name,
		Status:          "success",
		Edited:          false,
		Mentions:        originalMsg.Mentions,
		Media:           originalMsg.Media,
		OrgansationID:   channels.OrganisationID,
		IsForwarded:     true,
		ForwardedFromID: req.ThreadId,
		SenderID:        req.UserId,
	}

	if err := threadDoc.CreateThread(db, logger); err != nil {
		return nil, err
	}

	feed := models.FeedMessageRequest{
		ChannelID:   req.ChannelsId,
		UserName:    profile.UserName,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		AvatarURL:   profile.AvatarURL,
		Type:        "message",
		Content:     originalMsg.Content,
		ThreadId:    threadDoc.ID,
		Email:       user.Email,
		UserType:    originalMsg.UserType,
		FullName:    profile.FullName,
		UserId:      userID,
		OrgId:       channels.OrganisationID,
		Media:       originalMsg.Media,
		ChannelName: channels.Name,
	}

	if err := centrifuge.PublishChannel(logger, req.ChannelsId, feed); err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, error: %v", req.ChannelsId, err.Error()))
		return nil, fmt.Errorf("failed to publish thread data")
	}

	dataByte, _ := json.Marshal(feed)

	notifRec := models.PushNotificationRecord{
		ChannelType: models.Channel,
		Data:        string(dataByte),
		Sent:        false,
		ChannelId:   req.ChannelsId,
		Section:     models.ThreadSection,
		Type:        models.NewMessage,
	}

	if err := actions.AddPushNotificationToQueue(storage.DB.Redis, notifRec); err != nil {
		logger.Error("Error adding notification to channelid: %s, with orgid: %s error: %v", req.ChannelsId, channels.OrganisationID, err.Error())
	}

	logger.Info("added notification to queue for channel %s", req.ChannelsId)

	// increase unread count for channel users
	userChan.ChannelsID = req.ChannelsId
	userChan.UserID = userID
	userChan.OrgId = channels.OrganisationID
	var wg sync.WaitGroup
	mutex := &sync.Mutex{}

	// Add to the wait group for each goroutine that must complete first
	wg.Add(1)
	go func() {
		defer wg.Done()
		userChan.UpdateUnReadCount(db.Postgresql, mutex, logger)
	}()

	if len(originalMsg.Mentions) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			userChan.ProcessMentions(db.Postgresql, originalMsg.Mentions, mutex, logger)
		}()
	}

	// Run this after the others finish
	go func() {
		wg.Wait()
		userChan.SendChannelUnReadUpdate(mutex, logger, models.NewThread)
	}()

	return &threadDoc, nil
}

// 	messageDoc := models.MessageDocument{
// 		ID:               utility.GenerateUUID(),
// 		Content:          originalMsg.Content,
// 		ChannelsID:       req.ChannelsId,
// 		UserID:           userID,
// 		ThreadID:         threadId,
// 		AgentMessage:     originalMsg.AgentMessage,
// 		CreatedAt:        time.Now().UTC(),
// 		UpdatedAt:        time.Now().UTC(),
// 		AvatarURL:        profile.AvatarURL,
// 		Edited:           false,
// 		UserType:         originalMsg.UserType,
// 		Username:         profile.UserName,
// 		FullName:         profile.FullName,
// 		Email:            user.Email,
// 		Media:            originalMsg.Media,
// 		Mentions:         originalMsg.Mentions,
// 		OrganisationID:   channels.OrganisationID,
// 		IsForwarded:      true,
// 		ForwardedFromID:  req.MessageId,
// 		OriginalSenderID: req.UserId,
// 	}

// 	updateResp, err := messageDoc.CreateMessage(db, logger)
// 	if err != nil {
// 		return nil, errors.New("failed to save message, error: " + err.Error())
// 	}

// 	if err := thread.DetectAndAddMentions(messageDoc.ID, originalMsg.Content, db.Postgresql); err != nil {
// 		return &messageDoc, err
// 	}

// 	feed := models.FeedMessageRequest{
// 		ChannelID: req.ChannelsId,
// 		CreatedAt: messageDoc.CreatedAt.String(),
// 		UpdatedAt: messageDoc.UpdatedAt.String(),
// 		AvatarURL: profile.AvatarURL,
// 		Type:      "message",
// 		Content:   messageDoc.Content,
// 		ThreadId:  req.ThreadId,
// 		Email:     user.Email,
// 		UserType:  messageDoc.UserType,
// 		UserName:  user.Profile.UserName,
// 		FullName:  user.Profile.FullName,
// 		OrgId:     messageDoc.OrganisationID,
// 		UserId:    messageDoc.UserID,
// 		Media:     messageDoc.Media,
// 		Id:        messageDoc.ID,
// 	}

// 	err = centrifuge.PublishChannel(logger, threadId.String(), feed)
// 	if err != nil {
// 		logger.Error(fmt.Sprintf("Error Publishing to threadId: %s, error: %v", threadId.String(), err.Error()))
// 		return nil, errors.New("failed to publish webhook data: " + err.Error())
// 	}

// 	notification := models.Notification[models.ReplyCountChange]
// 	notification.SectionType = models.ChannelsSection
// 	notification.Content = feed
// 	notification.UpdateChange = updateResp

// 	err = centrifuge.PublishChannel(logger, req.ChannelsId, notification)
// 	if err != nil {
// 		logger.Error(fmt.Sprintf("Error Publishing forwarded message with destination id: %s error: %v", req.ChannelsId, err.Error()))
// 		return nil, errors.New("failed to publish data: " + err.Error())
// 	}

// 	dataByte, _ := json.Marshal(feed)

// 	notifRec := models.PushNotificationRecord{
// 		ChannelType:  models.Channel,
// 		Data:         string(dataByte),
// 		Sent:         false,
// 		ChannelId:    req.ChannelsId,
// 		Section:      models.ThreadSection,
// 		UpdateChange: updateResp,
// 		Type:         models.NewMessage,
// 	}

// 	if err = actions.AddPushNotificationToQueue(storage.DB.Redis, notifRec); err != nil {
// 		logger.Error("Error adding notification to channelid: %s, with orgid: %s error: %v", req.ChannelsId, channels.OrganisationID, err.Error())
// 	}

// 	logger.Info("added notification to queue for channel %s", req.ChannelsId)

// 	return &messageDoc, nil
// }
