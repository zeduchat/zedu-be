package dm

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gofrs/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	push_notifications "github.com/hngprojects/telex_be/services/pushNotifications"
	"github.com/hngprojects/telex_be/services/thread"
	"github.com/hngprojects/telex_be/utility"
)

// Reply message fn (in dm / group_dm)
func SaveChannelsDmMsg(req models.CreateMessageRequest, db *storage.Database, logger *utility.Logger) (*models.MessageDocument, int, error) {
	var (
		profile    models.Profile
		user       models.User
		channel    models.DmChannels
		channelIDs []string
		chanParts  []models.ChannelParticipant
	)

	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("invalid thread ID")
	}

	err = profile.GetProfileByUserId(db.Postgresql, req.UserId)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to get user profile")
	}

	user, err = user.GetUserByID(db.Postgresql, req.UserId)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to get user")
	}

	ch, err := channel.CheckChannelExists(db.Postgresql, req.ChannelsId)
	if !ch || err != nil {
		return nil, http.StatusNotFound, errors.New("channel does not exist")
	}

	messageDoc := models.MessageDocument{
		ID:             utility.GenerateUUID(),
		Content:        req.Content,
		ChannelsID:     req.ChannelsId,
		UserID:         req.UserId,
		ThreadID:       threadId,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		AvatarURL:      profile.AvatarURL,
		Edited:         false,
		Username:       profile.UserName,
		FullName:       profile.FullName,
		Email:          user.Email,
		OrganisationID: channel.OrgId,
		Mentions:       req.Mentions,
		Media:          req.Media,
	}

	err = messageDoc.CreateMessage(db, logger)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to save message, error: " + err.Error())
	}

	if err := thread.DetectAndAddMentions(messageDoc.ID, req.Content, db.Postgresql); err != nil {
		return &messageDoc, http.StatusBadRequest, err
	}

	feed := models.FeedMessageRequest{
		ChannelID: req.ChannelsId,
		UserName:  profile.UserName,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		AvatarURL: profile.AvatarURL,
		Type:      "message",
		Content:   req.Content,
		ThreadId:  req.ThreadId,
		Email:     user.Email,
		FullName:  profile.FullName,
		OrgId:     req.OrgId,
		UserId:    req.UserId,
		Media:     req.Media,
	}

	err = centrifuge.PublishChannel(logger, threadId.String(), feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to threadId: %s, error: %v", threadId.String(), err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to publish webhook data: " + err.Error())
	}

	notification := models.Notification[models.NewMessage]
	notification.SectionType = models.ReplySection
	notification.Content = feed

	username := ""
	if profile.UserName != "" {
		username = profile.UserName
	} else if profile.FullName != "" {
		username = profile.FullName
	} else {
		username = user.Email
	}

	err = postgresql.SelectAllFromDb(db.Postgresql, "", &chanParts, "channel_id = ?", channel.ChannelId)
	if err != nil {
		return &messageDoc, http.StatusNotFound, fmt.Errorf("failed to fetch participants")
	}

	for _, participant := range chanParts {
		if participant.UserId != req.UserId {
			channelIDs = append(channelIDs, participant.UserId)
		}
	}

	// Handle DM-specific case
	if channel.ChannelType == "dm" && len(channelIDs) == 1 {
		err = centrifuge.PublishChannel(logger, channelIDs[0], notification)
		if err != nil {
			logger.Error(fmt.Sprintf("Error Publishing to participant id: %s, error: %v", channelIDs[0], err.Error()))
			return nil, http.StatusBadRequest, errors.New("failed to publish webhook data: " + err.Error())
		}

		pushReq := models.PushFCMRequest{
			ChannelName: username,
			UserId:      channelIDs[0],
			Message:     req.Content,
			TimeStamp:   messageDoc.CreatedAt.String(),
			AvatarUrl:   profile.AvatarURL,
		}

		err = push_notifications.PushFCMToUser(pushReq, logger, db.Postgresql)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to send push notification to user %s: %v", channelIDs[0], err))
		}

		return &messageDoc, http.StatusCreated, nil
	}

	// Handle group DM case
	if len(channelIDs) > 0 {
		// Broadcast to all participants
		err = centrifuge.BatchBroadcastToChannel(logger, channelIDs, notification)
		if err != nil {
			logger.Error(fmt.Sprintf("Error Broadcasting to channel IDs: %v, error: %v", channelIDs, err.Error()))
			return nil, http.StatusInternalServerError, errors.New("failed to broadcast webhook data: " + err.Error())
		}

		// Send push notifications to all participants using the sender's username
		for _, userID := range channelIDs {
			pushReq := models.PushFCMRequest{
				ChannelName: username,
				UserId:      userID,
				Message:     req.Content,
				TimeStamp:   messageDoc.CreatedAt.String(),
				AvatarUrl:   profile.AvatarURL,
			}

			err = push_notifications.PushFCMToUser(pushReq, logger, db.Postgresql)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to send push notification to user %s: %v", userID, err))
			}
		}
	}

	return &messageDoc, http.StatusCreated, nil
}

func DeleteChannelsDmMsg(req models.EditMessageRequest) (*models.Message, int, error) {

	var message models.Message

	message.ID = req.MessageId

	if _, err := message.DeleteMessage(); err != nil {
		return nil, http.StatusBadRequest, err
	}

	return nil, http.StatusOK, nil
}

// Reply message fn
func AddChannelsDmMsg(req models.CreateMessageRequest, db *storage.Database,
	logger *utility.Logger) (*models.MessageDocument, int, error) {

	// Provision for bot dms

	return SaveChannelsDmMsg(req, db, logger)

}
