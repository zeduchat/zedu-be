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
		userChan    models.UserChannels
		dmChannel   models.DmChannels
	)

	chanExist := postgresql.CheckExists(db.Postgresql, &channels, "id = ?", req.ChannelsId)
	dmChanExists, _ := dmChannel.CheckChannelExists(db.Postgresql, req.ChannelsId, req.UserId)

	if !(chanExist || dmChanExists) {
		return nil, errors.New("channel does not exist")
	}

	if err := profile.GetProfileByUserId(db.Postgresql, userID); err != nil {
		return nil, errors.New("failed to get user profile")
	}

	user, userErr := user.GetUserByID(db.Postgresql, userID)
	if userErr != nil {
		return nil, errors.New("failed to get user")
	}

	if err := originalMsg.GetThreadById(db.Postgresql, req.ThreadId); err != nil {
		return nil, errors.New("failed to get thread message details")
	}

	// Create pair room if first message and not a bot
	thread := models.ThreadDocument{
		UserId:     req.UserId,
		ChannelsID: req.ChannelsId,
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
		ID:                     utility.GenerateUUID(),
		Username:               profile.UserName,
		Content:                originalMsg.Content,
		ChannelsID:             req.ChannelsId,
		Type:                   originalMsg.Type,
		MessageCount:           0,
		AvatarURL:              profile.AvatarURL,
		FullName:               profile.FullName,
		Email:                  user.Email,
		CreatedAt:              time.Now().UTC(),
		CurrentStatus:          "pending",
		UserType:               originalMsg.UserType,
		UserId:                 userID,
		Messages:               originalMsg.Messages,
		ChannelName:            utility.ThisOrThat(channels.Name, profile.FullName),
		Status:                 "success",
		Edited:                 originalMsg.Edited,
		Mentions:               originalMsg.Mentions,
		Media:                  originalMsg.Media,
		OrgansationID:          utility.ThisOrThat(channels.OrganisationID, dmChannel.OrgId),
		IsForwarded:            true,
		ForwardedFromID:        originalMsg.ID,
		ForwardedFromType:      originalMsg.Type,
		ForwardedFromChannelID: originalMsg.ChannelsID,
		ForwardedCreatedAt:     originalMsg.CreatedAt,
		SenderID:               originalMsg.UserId,
		SenderFullName:         originalMsg.FullName,
		SenderUsername:         originalMsg.Username,
		SenderAvatarURL:        originalMsg.AvatarURL,
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
		OrgId:       utility.ThisOrThat(channels.OrganisationID, dmChannel.OrgId),
		Media:       originalMsg.Media,
		ChannelName: utility.ThisOrThat(channels.Name, profile.FullName),
	}

	if err := centrifuge.PublishChannel(logger, req.ChannelsId, feed); err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, error: %v", req.ChannelsId, err.Error()))
		return nil, fmt.Errorf("failed to publish thread data")
	}

	dataByte, _ := json.Marshal(feed)
	typeChannelId := req.ChannelsId
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
		logger.Error("Error adding notification to channel Id: %s, with orgid: %s error: %v", req.ChannelsId, utility.ThisOrThat(channels.OrganisationID, dmChannel.OrgId), err.Error())
	}

	logger.Info("added notification to queue for channel %s", req.ChannelsId)

	if dmChanExists {
		dmChan := models.DmChannels{}
		dmChan.ChannelId = req.ChannelsId
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

	} else {
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
	}

	return &threadDoc, nil
}
