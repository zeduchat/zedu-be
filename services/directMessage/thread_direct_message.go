package dm

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	push_notifications "github.com/hngprojects/telex_be/services/pushNotifications"
	"github.com/hngprojects/telex_be/services/user"
	"github.com/hngprojects/telex_be/utility"
)

func SaveThreadDmMessage(req models.CreateThreadMsgReq, db *storage.Database, logger *utility.Logger) (*models.ThreadDocument, int, error) {
	var (
		profile    models.Profile
		user       models.User
		channel    models.DmChannels
		chanParts  []models.ChannelParticipant
		channelIDs []string // Renamed channelsIDs to channelIDs for consistency
	)

	err := profile.GetProfileByUserId(db.Postgresql, req.UserId)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to get user profile")
	}

	user, err = user.GetUserByID(db.Postgresql, req.UserId)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to get user")
	}

	exists, err := channel.CheckChannelExists(db.Postgresql, req.ChannelsID)
	if !exists || err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("channel does not exist: %v", err)
	}

	threadDoc := models.ThreadDocument{
		ID:            utility.GenerateUUID(),
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
		Status:        "success",
		Edited:        false,
		OrgansationID: req.OrgId,
	}

	err = threadDoc.CreateThread(db, logger)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to create thread: %v", err)
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
	}

	err = centrifuge.PublishChannel(logger, req.ChannelsID, feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, error: %v", req.ChannelsID, err))
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to publish webhook data: %v", err)
	}

	notification := models.Notification[models.NewMessage]
	notification.SectionType = models.ThreadSection
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
		return &threadDoc, http.StatusNotFound, fmt.Errorf("failed to fetch participants: %v", err)
	}

	for _, cp := range chanParts {
		if cp.UserId != req.UserId {
			channelIDs = append(channelIDs, cp.UserId)
		}
	}

	if len(channelIDs) == 0 {
		return &threadDoc, http.StatusCreated, nil
	}

	// Handle DM-specific case
	if channel.ChannelType == "dm" && len(channelIDs) == 1 {
		err = centrifuge.PublishChannel(logger, channelIDs[0], notification)
		if err != nil {
			logger.Error(fmt.Sprintf("Error Publishing to participant id: %s, error: %v", channelIDs[0], err))
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to publish webhook data: %v", err)
		}

		pushReq := models.PushFCMRequest{
			ChannelName: username,
			UserId:      channelIDs[0],
			Message:     req.Content,
			TimeStamp:   threadDoc.CreatedAt.String(),
			AvatarUrl:   profile.AvatarURL,
		}

		err = push_notifications.PushFCMToUser(pushReq, logger, db.Postgresql)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to send push notification to user %s: %v", channelIDs[0], err))
		}

		return &threadDoc, http.StatusCreated, nil
	}

	// Handle non-DM (group) case
	err = centrifuge.BatchBroadcastToChannel(logger, channelIDs, notification)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Broadcasting to channel IDs: %v, error: %v", channelIDs, err))
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to broadcast webhook data: %v", err)
	}

	pushFailures := 0
	for _, userID := range channelIDs {
		pushReq := models.PushFCMRequest{
			ChannelName: username,
			UserId:      userID,
			Message:     req.Content,
			TimeStamp:   threadDoc.CreatedAt.String(),
			AvatarUrl:   profile.AvatarURL,
		}

		err = push_notifications.PushFCMToUser(pushReq, logger, db.Postgresql)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to send push notification to user %s: %v", userID, err))
			pushFailures++
		}
	}
	if pushFailures > 0 {
		logger.Error(fmt.Sprintf("Failed to send push notifications to %d/%d users", pushFailures, len(channelIDs)))
	}

	return &threadDoc, http.StatusCreated, nil
}

// main channel thread
func CreateThreadDmMessage(req models.CreateThreadMsgReq, db *storage.Database, logger *utility.Logger) (*models.ThreadDocument, int, error) {

	// Provision for bot dms later

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

		res, err := dmchannel.CheckChannelExists(db.Postgresql, req.ChannelsID)

		if !res || err != nil {
			return &thread, http.StatusBadRequest, err
		}

		req.OrgId = dmchannel.OrgId

		if dmchannel.ChatType != "bot" {

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
