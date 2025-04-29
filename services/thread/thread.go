package thread

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/user"
	"github.com/hngprojects/telex_be/utility"
)

func CalculateStartTime(rangeStr string) time.Time {
	now := time.Now()
	switch strings.ToLower(rangeStr) {
	case "day":
		return now.Add(-24 * time.Hour)
	case "week":
		return now.Add(-7 * 24 * time.Hour)
	case "month":
		return now.AddDate(0, -1, 0)
	default:
		return time.Time{}
	}
}

func GetGroupByDate(c *gin.Context) (time.Time, bool) {

	if c.Query("groupBy") != "" {
		return CalculateStartTime(c.Query("groupBy")), true
	}

	return time.Time{}, false
}

func GetAllUserOrgThreads(orgID string, db *gorm.DB, c *gin.Context) (*[]models.Threads, *elastic.PaginationResponse, int, error) {
	var (
		accessData models.Threads
		accessResp []models.Threads
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

	accessResp, paginationResponse, err := accessData.GetUserThreadsByOrganization(c, db, userID, orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &accessResp, nil, http.StatusNoContent, nil
		}
		return &accessResp, nil, http.StatusInternalServerError, err

	}

	return &accessResp, paginationResponse, http.StatusOK, nil
}

func GetAllChannelThreads(channelID string, db *gorm.DB, c *gin.Context, logger *utility.Logger) ([]models.Threads, *elastic.PaginationResponse, int, error) {
	var (
		accessData         models.Threads
		accessResp         []models.Threads
		paginationResponse *elastic.PaginationResponse
		userChannel        models.UserChannels
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

	timeRange, check := GetGroupByDate(c)

	if check {
		accessResp, paginationResponse, err = accessData.GetAllGroupThreadsByChannelID(c, db, channelID, timeRange)

	} else {
		accessResp, paginationResponse, err = accessData.GetAllThreadsByChannelID(c, db, userID, channelID)
	}

	if err != nil {
		return accessResp, nil, http.StatusInternalServerError, err
	}

	if len(accessResp) > 0 {
		updateLastRead := models.UpdateLastRead{
			LastReadAt:   accessResp[0].CreatedAt,
			LastThreadId: accessResp[0].ID,
		}
		userChannel.ChannelsID = channelID
		userChannel.UserID = userID
		go userChannel.UpdateLastRead(db, updateLastRead, &sync.Mutex{}, logger)
	}

	return accessResp, paginationResponse, http.StatusOK, nil
}

func GetChannelThreads(channelID string, db *gorm.DB, c *gin.Context, logger *utility.Logger) ([]models.Threads, *elastic.PaginationResponse, int, error) {
	var (
		accessData models.Threads
		accessResp []models.Threads
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

	accessResp, paginationResponse, err := accessData.GetThreadsByChannelID(c, db, userID, channelID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return accessResp, nil, http.StatusNoContent, nil
		}
		return accessResp, nil, http.StatusInternalServerError, err

	}

	return accessResp, paginationResponse, http.StatusOK, nil
}

func GetUserSingleThreads(threadID, channelID string, db *gorm.DB, c *gin.Context) (*[]models.MessageDocument, *elastic.PaginationResponse, int, error) {
	var (
		messages models.Message
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

	accessResp, paginationResponse, err := messages.GetAllMessagesByThreadID(c, db, userID, threadID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &accessResp, nil, http.StatusNoContent, nil
		}
		return &accessResp, nil, http.StatusInternalServerError, err

	}

	return &accessResp, paginationResponse, http.StatusOK, nil
}

func UpdateAThread(req models.UpdateThreadStatus, threadID, channelID string, db *gorm.DB, c *gin.Context) (int, error) {
	var (
		thread models.Threads
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return http.StatusNotFound, err
	}

	userID, ok := userId.(string)
	if !ok {
		return http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	_, code, err := user.GetUser(userID, db)
	if err != nil {
		return code, err
	}

	thread.ID = threadID

	updateKey := map[string]interface{}{
		"current_status": req.Status,
	}

	if _, err := thread.UpdateThread(db, updateKey); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func DeleteAThread(threadID, channelID string, db *gorm.DB, c *gin.Context, logger *utility.Logger) (int, error) {
	var (
		thread     models.Threads
		threadDoc  models.ThreadDocument
		channel    models.Channels
		dmChannel  models.DmChannels
		publishDst string
		chanParts  []models.ChannelParticipant
		channelIDs []string
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return http.StatusNotFound, err
	}

	userID, ok := userId.(string)
	if !ok {
		return http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	_, code, err := user.GetUser(userID, db)
	if err != nil {
		return code, err
	}

	chanExist, _ := channel.CheckChannelExists(db, channelID)
	dmChanExist, _ := dmChannel.CheckChannelExists(db, channelID)

	if !(dmChanExist || chanExist) {
		return http.StatusNotFound, errors.New("channel does not exist")
	}

	thread.ID = threadID

	err = threadDoc.GetThreadById(db, threadID)
	if err != nil {
		return http.StatusNotFound, errors.New("thread not found")
	}

	if _, err := thread.DeleteThread(db); err != nil {
		return http.StatusBadRequest, err
	}

	notification := models.Notification[models.Deleted]
	notification.SectionType = models.ThreadSection
	notification.ModifcationDetails = models.ModifcationDetails{
		ThreadId:  threadID,
		ChannelId: channelID,
	}

	if channel.OrganisationID != "" {
		publishDst = channel.OrganisationID
	} else {
		if dmChannel.ChannelType == "dm" {
			publishDst = *dmChannel.ParticipantId
		}
	}

	if dmChannel.ChannelType == "group_dm" && channel.OrganisationID == "" {
		err := postgresql.SelectAllFromDb(db, "", &chanParts, "channel_id = ?", dmChannel.ChannelId)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to fetch channel participants: %s", err)
		}

		for _, participant := range chanParts {
			if participant.UserId != userID {
				channelIDs = append(channelIDs, participant.UserId)
			}
		}

		if len(channelIDs) == 0 {
			return http.StatusOK, nil
		}

		err = centrifuge.BatchBroadcastToChannel(logger, channelIDs, notification)
		if err != nil {
			logger.Error(fmt.Sprintf("Error broadcasting to with destination id: %s error: %v", publishDst, err.Error()))
			return http.StatusBadRequest, errors.New("failed to broadcast data")
		}

		return http.StatusOK, nil
	}

	if dmChannel.ChannelType == "dm" {
		err = centrifuge.BatchBroadcastToChannel(logger, []string{publishDst, userID}, notification)
		if err != nil {
			logger.Error(fmt.Sprintf("Error Publishing to with destination id: %s error: %v", publishDst, err.Error()))
			return http.StatusBadRequest, errors.New("failed to publish data")
		}
		return http.StatusOK, nil
	}

	err = centrifuge.PublishChannel(logger, publishDst, notification)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to with destination id: %s error: %v", publishDst, err.Error()))
		return http.StatusBadRequest, errors.New("failed to publish data")
	}

	return http.StatusOK, nil
}

func UpdateThreadMessage(req models.UpdateThreadMessage, db *gorm.DB, c *gin.Context, logger *utility.Logger) (models.ThreadDocument, int, error) {
	var (
		thread     models.Threads
		threadResp models.ThreadDocument
		publishDst string
		user       models.User
		dmChannel  models.DmChannels
		channel    models.Channels
		chanParts  []models.ChannelParticipant
		channelIDs []string
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return threadResp, http.StatusNotFound, err
	}

	userID, ok := userId.(string)
	if !ok {
		return threadResp, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	chanExist, _ := channel.CheckChannelExists(db, req.ChannelId)
	dmChanExist, _ := dmChannel.CheckChannelExists(db, req.ChannelId)

	if !(dmChanExist || chanExist) {
		return threadResp, http.StatusNotFound, errors.New("channel does not exist")
	}

	user, err = user.GetUserByID(db, userID)

	if err != nil {
		return threadResp, http.StatusBadRequest, errors.New("failed to get user")
	}

	thread.ID = req.ThreadId

	updateKey := map[string]interface{}{
		"message": req.Message,
		"edited":  true,
	}

	if _, err := thread.UpdateThread(db, updateKey); err != nil {
		return threadResp, http.StatusNotFound, err
	}

	if channel.OrganisationID != "" {
		publishDst = channel.OrganisationID
	} else {
		if dmChannel.ChannelType == "dm" {
			publishDst = *dmChannel.ParticipantId
		}
	}

	err = threadResp.GetThreadById(db, thread.ID)

	if err != nil {
		return threadResp, http.StatusInternalServerError, err
	}

	feed := models.FeedMessageRequest{
		ChannelID: threadResp.ChannelsID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		AvatarURL: threadResp.AvatarURL,
		Type:      threadResp.Type,
		Content:   threadResp.Content,
		ThreadId:  req.ThreadId,
		Email:     threadResp.Email,
		UserType:  threadResp.UserType,
		UserName:  user.Profile.UserName,
		FullName:  user.Profile.FullName,
		OrgId:     threadResp.OrgansationID,
		UserId:    threadResp.UserId,
		Media:     threadResp.Media,
	}

	notification := models.Notification[models.Updated]
	notification.SectionType = models.ThreadSection
	notification.Content = feed
	notification.ModifcationDetails = models.ModifcationDetails{
		ThreadId:  req.ThreadId,
		ChannelId: req.ChannelId,
	}

	if dmChannel.ChannelType == "group_dm" && channel.OrganisationID == "" {
		err := postgresql.SelectAllFromDb(db, "", &chanParts, "channel_id = ?", dmChannel.ChannelId)
		if err != nil {
			return threadResp, http.StatusInternalServerError, fmt.Errorf("failed to fetch channel participants: %s", err)
		}

		for _, participant := range chanParts {
			if participant.UserId != userID {
				channelIDs = append(channelIDs, participant.UserId)
			}
		}

		if len(channelIDs) == 0 {
			return threadResp, http.StatusOK, nil
		}

		err = centrifuge.BatchBroadcastToChannel(logger, channelIDs, notification)
		if err != nil {
			logger.Error(fmt.Sprintf("Error broadcasting to with destination id: %s error: %v", publishDst, err.Error()))
			return threadResp, http.StatusBadRequest, errors.New("failed to broadcast data")
		}

		return threadResp, http.StatusOK, nil
	}

	if dmChannel.ChannelType == "dm" {

		err = centrifuge.BatchBroadcastToChannel(logger, []string{publishDst, userID}, notification)
		if err != nil {
			logger.Error(fmt.Sprintf("Error Publishing to with destination id: %s error: %v", publishDst, err.Error()))
			return threadResp, http.StatusBadRequest, errors.New("failed to publish data")
		}
		return threadResp, http.StatusOK, nil
	}

	err = centrifuge.PublishChannel(logger, publishDst, notification)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to with destination id: %s error: %v", publishDst, err.Error()))
		return threadResp, http.StatusBadRequest, errors.New("failed to publish data")
	}

	return threadResp, http.StatusOK, nil
}

func ChannelCountInfo(c *gin.Context, db *storage.Database, org_id string, days int) (models.ChannelCountInfo, []models.ChannelMetrics, error) {
	var (
		channel models.ChannelCountInfo
		t       models.Threads
		cm      []models.ChannelMetrics
	)

	response, channelInfoMetrics, err := t.GetChannelCountInfo(db, org_id, days)
	if err != nil {
		return channel, cm, err
	}
	return response, channelInfoMetrics, nil
}

func PostFeedMessage(db *storage.Database, logger *utility.Logger, req models.CreateThreadMsgReq) (models.ThreadDocument, int, error) {

	thread_doc, err := SaveThreadMessage(req, db, logger)
	if err != nil {
		logger.Error("failed to create message thread" + err.Error())
		return models.ThreadDocument{}, http.StatusInternalServerError, err
	}

	return *thread_doc, http.StatusOK, nil
}
