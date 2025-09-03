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

func ForwardReplyMessage(db *storage.Database, req models.ForwardReplyMessageRequest, logger *utility.Logger, userID string) (*models.ThreadDocument, error) {
	var (
		originalMsg          models.MessageDocument
		user                 models.User
		profile              models.Profile
		channels             models.Channels
		fwdChannels          models.Channels
		dmChannels           models.DmChannels
		channelToForwardToID = req.ForwardedToChannelId.String()
	)

	chanExist := postgresql.CheckExists(db.Postgresql, &channels, "id = ?", req.ChannelsId)
	dmChanExist, _ := dmChannels.CheckChannelExists(db.Postgresql, req.ChannelsId, userID)
	if !(chanExist || dmChanExist) {
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
		return nil, errors.New("failed to get reply message details")
	}

	fwdChanExist := postgresql.CheckExists(db.Postgresql, &fwdChannels, "id = ?", channelToForwardToID)
	fwdDmChanExist, _ := dmChannels.CheckChannelExists(db.Postgresql, channelToForwardToID, userID)
	if !(fwdChanExist || fwdDmChanExist) {
		return nil, errors.New("forwarded channel does not exist")
	}

	if fwdChanExist {
		threadDocument, err := ForwardReplyMessageToChannel(db, originalMsg, req, logger, user, profile, fwdChannels)
		if err != nil {
			return nil, err
		}

		return threadDocument, nil
	} else {
		threadDocument, err := ForwardReplyMessageToDmChannel(db, originalMsg, req, logger, user, profile, dmChannels)
		if err != nil {
			return nil, err
		}

		return threadDocument, nil
	}
}

func ForwardReplyMessageToDmChannel(db *storage.Database, originalMsg models.MessageDocument, req models.ForwardReplyMessageRequest, logger *utility.Logger, user models.User, profile models.Profile, dmChannels models.DmChannels) (*models.ThreadDocument, error) {
	var (
		channelToForwardToID = req.ForwardedToChannelId.String()
		messageType          = "message"
		userType             = "user"
	)

	// Create pair room if first message and not a bot
	thread := models.ThreadDocument{
		UserId:     req.UserId,
		ChannelsID: channelToForwardToID,
	}

	pairRoom, _, _ := thread.CheckExists()
	if !pairRoom && dmChannels.ChatType != "bot" {
		pairRoomChan := models.DmChannels{}

		pairRoomChan.ChatType = dmChannels.ChatType
		pairRoomChan.ChannelType = "dm"
		pairRoomChan.UserId = *dmChannels.ParticipantId
		pairRoomChan.ParticipantId = &dmChannels.UserId
		pairRoomChan.ID = utility.GenerateUUID()
		pairRoomChan.ChannelId = dmChannels.ChannelId
		pairRoomChan.OrgId = dmChannels.OrgId

		_, err := pairRoomChan.CreateDmChannel(db.Postgresql)
		if err != nil {
			return &thread, err
		}
	}

	threadDoc := models.ThreadDocument{
		ID:            utility.GenerateUUID(),
		Username:      profile.UserName,
		Content:       req.Content,
		ChannelsID:    channelToForwardToID,
		Type:          messageType,
		MessageCount:  0,
		AvatarURL:     profile.AvatarURL,
		FullName:      profile.FullName,
		Email:         user.Email,
		CreatedAt:     time.Now().UTC(),
		CurrentStatus: "pending",
		UserType:      userType,
		UserId:        req.UserId,
		ChannelName:   profile.FullName,
		ChannelType:   "DM",
		Status:        "success",
		Mentions:      req.Mentions,
		Media:         req.Media,
		OrgansationID: dmChannels.OrgId,
		IsForwarded:   true,
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
		UserId:      req.UserId,
		OrgId:       dmChannels.OrgId,
		Media:       req.Media,
		ChannelName: profile.FullName,
		ChannelType: "DM",
	}

	if err := centrifuge.PublishChannel(logger, channelToForwardToID, feed); err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, error: %v", channelToForwardToID, err.Error()))
		return nil, fmt.Errorf("failed to publish thread data")
	}

	dataByte, _ := json.Marshal(feed)
	typeChannelId := *dmChannels.ParticipantId
	channelType := models.DMChannel

	notifRec := models.PushNotificationRecord{
		ChannelType: channelType,
		Data:        string(dataByte),
		Sent:        false,
		ChannelId:   typeChannelId,
		Section:     models.ThreadSection,
		Type:        models.NewMessage,
	}

	if err := actions.AddPushNotificationToQueue(storage.DB.Redis, notifRec); err != nil {
		logger.Error("Error adding notification to channelid: %s, with orgid: %s error: %v", channelToForwardToID, dmChannels.OrgId, err.Error())
	}

	logger.Info("added notification to queue for channel %s", req.ChannelsId)

	dmChan := models.DmChannels{}
	dmChan.ChannelId = channelToForwardToID
	dmChan.UserId = req.UserId
	dmChan.ChannelType = dmChannels.ChannelType
	dmChan.OrgId = dmChannels.OrgId

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

func ForwardReplyMessageToChannel(db *storage.Database, originalMsg models.MessageDocument, req models.ForwardReplyMessageRequest, logger *utility.Logger, user models.User, profile models.Profile, channels models.Channels) (*models.ThreadDocument, error) {
	var (
		userChan             models.UserChannels
		channelsType         string
		channelToForwardToID = req.ForwardedToChannelId.String()
		messageType          = "message"
		userType             = "user"
	)

	if channels.IsPrivate {
		channelsType = "private"
	} else {
		channelsType = "public"
	}

	threadDoc := models.ThreadDocument{
		ID:            utility.GenerateUUID(),
		Username:      profile.UserName,
		Content:       req.Content,
		ChannelsID:    channelToForwardToID,
		Type:          messageType,
		MessageCount:  0,
		AvatarURL:     profile.AvatarURL,
		FullName:      profile.FullName,
		Email:         user.Email,
		CreatedAt:     time.Now().UTC(),
		CurrentStatus: "pending",
		UserType:      userType,
		UserId:        req.UserId,
		ChannelName:   channels.Name,
		ChannelType:   channelsType,
		Status:        "success",
		Edited:        originalMsg.Edited,
		Mentions:      req.Mentions,
		Media:         req.Media,
		OrgansationID: channels.OrganisationID,
		IsForwarded:   true,
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
		UserId:      req.UserId,
		OrgId:       channels.OrganisationID,
		Media:       req.Media,
		ChannelName: channels.Name,
		ChannelType: channelsType,
	}

	if err := centrifuge.PublishChannel(logger, channelToForwardToID, feed); err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, error: %v", channelToForwardToID, err.Error()))
		return nil, fmt.Errorf("failed to publish thread data")
	}

	dataByte, _ := json.Marshal(feed)
	typeChannelId := channelToForwardToID
	channelType := models.Channel

	notifRec := models.PushNotificationRecord{
		ChannelType: channelType,
		Data:        string(dataByte),
		Sent:        false,
		ChannelId:   typeChannelId,
		Section:     models.ThreadSection,
		Type:        models.NewMessage,
	}

	if err := actions.AddPushNotificationToQueue(storage.DB.Redis, notifRec); err != nil {
		logger.Error("Error adding notification to channelid: %s, with orgid: %s error: %v", channelToForwardToID, channels.OrganisationID, err.Error())
	}

	logger.Info("added notification to queue for channel %s", channelToForwardToID)

	// increase unread count for channel users
	userChan.ChannelsID = channelToForwardToID
	userChan.UserID = req.UserId
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
		userChan.SendChannelUnReadUpdate(mutex, logger, models.NewThread, models.MentionMessage{})
	}()

	return &threadDoc, nil
}
