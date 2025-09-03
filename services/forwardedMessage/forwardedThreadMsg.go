package forwardedMessage

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/utility"
)

func ForwardThreadMessage(db *storage.Database, req models.ForwardThreadMessageRequest, logger *utility.Logger, userID string) (*models.ThreadDocument, error) {
	var (
		originalMsg            models.ThreadDocument
		user                   models.User
		profile                models.Profile
		channels               models.Channels
		fwdChannels            models.Channels
		dmChannels             models.DmChannels
		channelToForwardToID   = req.ForwardedToChannelId.String()
		originalMsgChannelType string
	)

	chanExist := postgresql.CheckExists(db.Postgresql, &channels, "id = ?", req.ChannelsId)
	dmChanExists, _ := dmChannels.CheckChannelExists(db.Postgresql, req.ChannelsId, userID)
	if !(chanExist || dmChanExists) {
		return nil, errors.New("channel does not exist")
	}

	if dmChanExists {
		originalMsgChannelType = "DM"
	} else if channels.IsPrivate {
		originalMsgChannelType = "private"
	} else {
		originalMsgChannelType = "public"
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

	fwdChanExist := postgresql.CheckExists(db.Postgresql, &fwdChannels, "id = ?", channelToForwardToID)
	fwdDmChanExist, _ := dmChannels.CheckChannelExists(db.Postgresql, channelToForwardToID, userID)
	if !(fwdChanExist || fwdDmChanExist) {
		return nil, errors.New("forwarded channel does not exist")
	}

	if fwdChanExist {
		threadDoc, err := ForwardThreadMessageToChannel(db, req, logger, originalMsg, profile, user, fwdChannels, originalMsgChannelType)
		if err != nil {
			return nil, err
		}
		return threadDoc, nil
	} else {
		threadDoc, err := ForwardThreadMessageToDM(db, req, logger, originalMsg, profile, user, dmChannels, originalMsgChannelType)
		if err != nil {
			return nil, err
		}
		return threadDoc, nil
	}
}

func ForwardThreadMessageToChannel(db *storage.Database, req models.ForwardThreadMessageRequest, logger *utility.Logger, originalMsg models.ThreadDocument, profile models.Profile, user models.User, channel models.Channels, originalMsgChannelType string) (*models.ThreadDocument, error) {
	var (
		channelToForwardToID = req.ForwardedToChannelId.String()
		messageType          = "message"
		userType             = "user"
		orgId                = channel.OrganisationID
		channelType          string
		userChan             models.UserChannels
	)

	if channel.IsPrivate {
		channelType = "private"
	} else {
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
		Mentions:      req.Mentions,
		Media:         req.Media,
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
			OriginalContent:         originalMsg.Content,
			OriginalChannelType:     originalMsgChannelType,
			OriginalCreatedAt:       time.Now().UTC(),
			IsThread:                true,
		},
		OrgansationID: orgId,
	}
	err := threadDoc.CreateThread(db, logger)
	if err != nil {
		return nil, err
	}

	feed := models.FeedMessageRequest{
		ChannelID:   channelToForwardToID,
		UserName:    utility.ThisOrThat(profile.UserName, "a-user"),
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		AvatarURL:   profile.AvatarURL,
		Type:        messageType,
		Content:     req.Content,
		ThreadId:    threadDoc.ID,
		Email:       user.Email,
		UserType:    userType,
		FullName:    utility.ThisOrThat(profile.FullName, "a-user"),
		UserId:      req.UserId,
		OrgId:       channel.OrganisationID,
		ChannelName: channel.Name,
		ChannelType: channelType,
		IsForwarded: true,
		Media:       req.Media,
		ForwardedMessageMetadata: &models.ForwardedMessageMetadata{
			OriginalMessageID:       originalMsg.ID,
			OriginalSenderID:        originalMsg.UserId,
			OriginalSenderName:      originalMsg.FullName,
			OriginalSenderUsername:  originalMsg.Username,
			OriginalSenderAvatarURL: originalMsg.AvatarURL,
			OriginalChannelID:       originalMsg.ChannelsID,
			OriginalContent:         originalMsg.Content,
			OriginalChannelName:     originalMsg.ChannelName,
			OriginalChannelType:     originalMsgChannelType,
			OriginalCreatedAt:       time.Now().UTC(),
			IsThread:                true,
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
		userChan.SendChannelUnReadUpdate(mutex, logger, models.NewThread, models.MentionMessage{})
	}()

	return &threadDoc, nil
}

func ForwardThreadMessageToDM(db *storage.Database, req models.ForwardThreadMessageRequest, logger *utility.Logger, originalMsg models.ThreadDocument, profile models.Profile, user models.User, dmChannel models.DmChannels, originalMsgChannelType string) (*models.ThreadDocument, error) {
	var (
		channelToForwardToID = req.ForwardedToChannelId.String()
		messageType          = "message"
		userType             = "user"
		channelsType         = "DM"
	)

	// Create pair room if first message and not a bot
	thread := models.ThreadDocument{
		UserId:     req.UserId,
		ChannelsID: channelToForwardToID,
	}

	pairRoom, _, _ := thread.CheckExists()
	if !pairRoom && dmChannel.ChatType != "bot" && dmChannel.ChannelType == "dm" {
		pairRoomChan := models.DmChannels{}

		pairRoomChan.ChatType = dmChannel.ChatType
		pairRoomChan.ChannelType = "dm"
		pairRoomChan.UserId = *dmChannel.ParticipantId
		pairRoomChan.ParticipantId = &dmChannel.UserId
		pairRoomChan.ID = utility.GenerateUUID()
		pairRoomChan.ChannelId = dmChannel.ChannelId
		pairRoomChan.OrgId = dmChannel.OrgId

		_, err := pairRoomChan.CreateDmChannel(db.Postgresql)
		if err != nil {
			return &thread, err
		}
	}

	threadDoc := models.ThreadDocument{
		ID:            utility.GenerateUUID(),
		Username:      utility.ThisOrThat(profile.UserName, "a-user"),
		Content:       req.Content,
		ChannelsID:    channelToForwardToID,
		Type:          messageType,
		MessageCount:  0,
		Mentions:      req.Mentions,
		Media:         req.Media,
		AvatarURL:     profile.AvatarURL,
		FullName:      utility.ThisOrThat(profile.FullName, "a-user"),
		Email:         user.Email,
		CreatedAt:     time.Now().UTC(),
		CurrentStatus: "pending",
		UserType:      userType,
		UserId:        req.UserId,
		Messages:      []models.MessageDocument{},
		ChannelName:   profile.FullName,
		ChannelType:   channelsType,
		Status:        "success",
		Edited:        false,
		IsForwarded:   true,
		ForwardedMessageMetadata: &models.ForwardedMessageMetadata{
			OriginalMessageID:       originalMsg.ID,
			OriginalSenderID:        originalMsg.UserId,
			OriginalSenderName:      originalMsg.FullName,
			OriginalSenderUsername:  originalMsg.Username,
			OriginalContent:         originalMsg.Content,
			OriginalSenderAvatarURL: originalMsg.AvatarURL,
			OriginalChannelID:       originalMsg.ChannelsID,
			OriginalChannelName:     originalMsg.ChannelName,
			OriginalChannelType:     originalMsgChannelType,
			OriginalCreatedAt:       time.Now().UTC(),
			IsThread:                true,
		},
		OrgansationID: dmChannel.OrgId,
	}

	if err := threadDoc.CreateThread(db, logger); err != nil {
		return nil, err
	}

	feed := models.FeedMessageRequest{
		ChannelID:   channelToForwardToID,
		UserName:    profile.UserName,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		AvatarURL:   profile.AvatarURL,
		Type:        messageType,
		Content:     req.Content,
		ThreadId:    threadDoc.ID,
		Email:       user.Email,
		UserType:    userType,
		FullName:    profile.FullName,
		Media:       req.Media,
		UserId:      req.UserId,
		OrgId:       dmChannel.OrgId,
		ChannelName: profile.FullName,
		ChannelType: channelsType,
		IsForwarded: true,
		ForwardedMessageMetadata: &models.ForwardedMessageMetadata{
			OriginalMessageID:       originalMsg.ID,
			OriginalSenderID:        originalMsg.UserId,
			OriginalSenderName:      originalMsg.FullName,
			OriginalSenderUsername:  originalMsg.Username,
			OriginalSenderAvatarURL: originalMsg.AvatarURL,
			OriginalChannelID:       originalMsg.ChannelsID,
			OriginalChannelName:     originalMsg.ChannelName,
			OriginalChannelType:     originalMsgChannelType,
			OriginalContent:         originalMsg.Content,
			OriginalCreatedAt:       time.Now().UTC(),
			IsThread:                true,
		},
	}

	if err := centrifuge.PublishChannel(logger, channelToForwardToID, feed); err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, error: %v", channelToForwardToID, err.Error()))
		return nil, fmt.Errorf("failed to publish thread data")
	}

	dataByte, _ := json.Marshal(feed)
	typeChannelId := channelToForwardToID
	channelType := models.Channel

	if dmChannel.ChannelType == "dm" {
		typeChannelId = *dmChannel.ParticipantId
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

	if err := actions.AddPushNotificationToQueue(storage.DB.Redis, notifRec); err != nil {
		logger.Error("Error adding notification to channel Id: %s, with orgid: %s error: %v", channelToForwardToID, dmChannel.OrgId, err.Error())
	}

	logger.Info("added notification to queue for channel %s", channelToForwardToID)

	dmChan := models.DmChannels{}
	dmChan.ChannelId = channelToForwardToID
	dmChan.UserId = req.UserId
	dmChan.ChannelType = dmChannel.ChannelType
	dmChan.OrgId = dmChannel.OrgId

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

	return &threadDoc, nil
}
