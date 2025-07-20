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

func ForwardThreadMessage(db *storage.Database, req models.ForwardThreadMessageRequest, logger *utility.Logger, userID string) (*models.ThreadDocument, error) {
	var (
		originalMsg models.ThreadDocument
		user        models.User
		profile     models.Profile
		channels    models.Channels
		dmChannel   models.DmChannels
		threadDoc   models.ThreadDocument
	)

	chanExist := postgresql.CheckExists(db.Postgresql, &channels, "id = ?", req.ChannelsId)
	dmChanExists, _ := dmChannel.CheckChannelExists(db.Postgresql, req.ChannelsId, req.UserId)

	if !(chanExist || dmChanExists) {
		return nil, errors.New("channel does not exist")
	}

	if err := profile.GetProfileByUserId(db.Postgresql, userID); err != nil {
		return nil, err
	}

	user, userErr := user.GetUserByID(db.Postgresql, userID)
	if userErr != nil {
		return nil, userErr
	}

	if err := originalMsg.GetThreadById(db.Postgresql, req.ThreadId); err != nil {
		return nil, err
	}

	if req.ForwardedToChannelId != nil {
		ForwardThreadMessageToChannel(db, req, logger, originalMsg, profile, user, channels)
	}

	return &threadDoc, nil
}

func ForwardThreadMessageToChannel(db *storage.Database, req models.ForwardThreadMessageRequest, logger *utility.Logger, originalMsg models.ThreadDocument, profile models.Profile, user models.User, current_channel models.Channels) (*models.ThreadDocument, error) {
	var (
		channel              models.Channels
		channelToForwardToID = req.ForwardedToChannelId.String()
		messageType          = "message"
		userType             = "user"
		orgId                = current_channel.OrganisationID
		channelType          string
		userChan			 models.UserChannels
	)

	exists, err := channel.CheckChannelExists(db.Postgresql, channelToForwardToID)
	if !exists || err != nil {
		return nil, err
	}

	if channel.IsPrivate {
		channelType = "private"
	}else{
		channelType = "public"
	}

	threadDoc := models.ThreadDocument{
		ID:            utility.GenerateUUID(),
		Username:      utility.ThisOrThat(profile.UserName, "a-user"),
		Content:       req.Content,
		ChannelsID:    channelToForwardToID,
		Type:          messageType,
		MessageCount:  0,
		AvatarURL:     profile.AvatarURL,
		FullName:      utility.ThisOrThat(profile.FullName, "a-user"),
		Email:         user.Email,
		CreatedAt:     time.Now().UTC(),
		CurrentStatus: "pending",
		UserType:      userType,
		UserId:        req.UserId,
		Messages:      []models.MessageDocument{},
		ChannelName:   channel.Name,
		ChannelType:   channelType,
		Status:        "success",
		Edited:        false,
		IsForwarded:   true,
		ForwardedMessageMetadata: &models.ForwardedMessageMetadata{
			OriginalMessageID:       originalMsg.ID,
			OriginalSenderID:        originalMsg.UserId,
			OriginalSenderName:      originalMsg.FullName,
			OriginalSenderUsername:  originalMsg.Username,
			OriginalSenderAvatarURL: originalMsg.AvatarURL,
			OriginalChannelID:       originalMsg.ChannelsID,
			OriginalChannelName:     originalMsg.ChannelName,
			OriginalCreatedAt:       time.Now().UTC(),
			IsThread: true,
		},
		OrgansationID: orgId,
	}
	err = threadDoc.CreateThread(db, logger)
	if err != nil {
		return nil, err
	}

	feed := models.FeedMessageRequest{
		ChannelID:   channelToForwardToID,
		UserName:    utility.ThisOrThat(profile.UserName, "a-user"),
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		AvatarURL:   profile.AvatarURL,
		Type:        "message",
		Content:     req.Content,
		ThreadId:    threadDoc.ID,
		Email:       user.Email,
		UserType:    userType,
		FullName:    utility.ThisOrThat(profile.FullName, "a-user"),
		UserId:      req.UserId,
		OrgId:       channel.OrganisationID,
		ChannelName: channel.Name,
		IsForwarded: true,
		ForwardedMessageMetadata: &models.ForwardedMessageMetadata{
			OriginalMessageID:       originalMsg.ID,
			OriginalSenderID:        originalMsg.UserId,
			OriginalSenderName:      originalMsg.FullName,
			OriginalSenderUsername:  originalMsg.Username,
			OriginalSenderAvatarURL: originalMsg.AvatarURL,
			OriginalChannelID:       originalMsg.ChannelsID,
			OriginalChannelName:     originalMsg.ChannelName,
			OriginalCreatedAt:       time.Now().UTC(),
			IsThread: true,
		},
	}

	err = centrifuge.PublishChannel(logger, channelToForwardToID, feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, error: %v", channelToForwardToID, err.Error()))
		return nil, fmt.Errorf("failed to publish thread data")
	}

	dataByte, _ := json.Marshal(feed)

	notifRec := models.PushNotificationRecord{
		ChannelType: models.Channel,
		Data:        string(dataByte),
		Sent:        false,
		ChannelId:   channelToForwardToID,
		Section:     models.ThreadSection,
		Type:        models.NewMessage,
	}

	err = actions.AddPushNotificationToQueue(storage.DB.Redis, notifRec)

	if err != nil {
		logger.Error("Error adding notification to channelid: %s, with orgid: %s error: %v", channelToForwardToID, orgId, err.Error())
	}

	logger.Info("added notification to queue for channel %s", channelToForwardToID)

	// increase unread count for channel users
	userChan.ChannelsID = channelToForwardToID
	userChan.UserID = req.UserId
	userChan.OrgId = channel.OrganisationID
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
		userChan.SendChannelUnReadUpdate(mutex, logger, models.NewThread)
	}()

	return &threadDoc, nil

}