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

	if err := originalMsg.GetThreadById(db.Postgresql, req.ThreadId); err != nil {
		return nil, errors.New("failed to get thread message details")
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
		ChannelName:            channels.Name,
		Status:                 "success",
		Edited:                 originalMsg.Edited,
		Mentions:               originalMsg.Mentions,
		Media:                  originalMsg.Media,
		OrgansationID:          channels.OrganisationID,
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
