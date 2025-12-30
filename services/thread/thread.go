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

func GetAllUserOrgThreads(orgID string, db *gorm.DB, c *gin.Context, logger *utility.Logger) (*[]models.ThreadWithMessagesResponse, *elastic.PaginationResponse, int, error) {
	var (
		accessData models.Threads
		accessResp []models.ThreadWithMessagesResponse
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

	accessData.UserId = userID
	accessData.OrgansationID = orgID

	accessResp, paginationResponse, err := accessData.GetUserThreadsByOrganization(c, db, logger)
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
		uc                 models.UserChannels
		dmChan             models.DmChannels
		channel            models.Channels
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
		accessResp, paginationResponse, code,  err = accessData.GetAllGroupThreadsByChannelID(c, db, channelID, timeRange)
	} else {
		accessResp, paginationResponse, code, err = accessData.GetAllThreadsByChannelID(c, db, userID, channelID)
	}

	if err != nil {
		return accessResp, nil, code, err
	}

	updateCall := map[string]func(chanType, orgId string){
		"channels": func(chanType, orgId string) {
			var userChannel models.UserChannels
			updateLastRead := models.UpdateLastRead{
				LastReadAt:   accessResp[0].CreatedAt,
				LastThreadId: accessResp[0].ID,
			}
			userChannel.ChannelsID = channelID
			userChannel.UserID = userID
			userChannel.OrgId = orgId
			var wg sync.WaitGroup

			wg.Add(1)

			go func() {
				defer wg.Done()
				updated := userChannel.UpdateLastRead(db, updateLastRead, &sync.Mutex{}, logger)
				if updated {
					userChannel.SendChannelUnReadUpdate(&sync.Mutex{}, logger, models.Read, models.MentionMessage{})
				}
			}()
		},
		"dm": func(chanType, orgId string) {
			var dmChan models.DmChannels
			updateLastRead := models.UpdateLastRead{
				LastReadAt:   accessResp[0].CreatedAt,
				LastThreadId: accessResp[0].ID,
			}
			dmChan.ChannelId = channelID
			dmChan.UserId = userID
			dmChan.ChannelType = chanType
			dmChan.OrgId = orgId
			var wg sync.WaitGroup

			wg.Add(1)

			go func() {
				defer wg.Done()
				updated := dmChan.UpdateLastRead(db, updateLastRead, &sync.Mutex{}, logger)
				if updated {
					dmChan.SendChannelUnReadUpdate(&sync.Mutex{}, logger, models.Read)
				}
			}()
		},
	}

	exists := postgresql.CheckExists(db, &uc, "channels_id = ?", channelID)
	if exists {

		ch, err := channel.CheckChannelExists(db, channelID)

		if !ch || err != nil {
			return nil, nil, http.StatusBadRequest, fmt.Errorf("channel does not exist: %v", err)
		}

		logger.Info("updating user channel count for channel %s", channelID)
		updateCall["channels"]("", channel.OrganisationID)
	} else if postgresql.CheckExists(db, &dmChan, "channel_id = ?", channelID) {

		logger.Info("updating user dm channel count for channel %s", channelID)
		updateCall["dm"](dmChan.ChannelType, dmChan.OrgId)
	} else {
		logger.Error("unable to update channel count for user, channel not found!, channelID: %s", channelID)
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

	updateKey := map[string]any{
		"current_status": req.Status,
	}

	if _, err := thread.UpdateThread(db, updateKey); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func DeleteAThread(threadID, channelID string, db *gorm.DB, c *gin.Context, logger *utility.Logger) (int, error) {
	var (
		thread       models.Threads
		threadDoc    models.ThreadDocument
		channel      models.Channels
		dmChannel    models.DmChannels
		savedMessage models.SavedMessage
	)

	// Start transaction
	tx := db.Begin()
	if tx.Error != nil {
		logger.Error("Failed to begin transaction: %v", tx.Error)
		return http.StatusInternalServerError, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	userId, err := middleware.GetUserClaims(c, tx, "user_id")
	if err != nil {
		tx.Rollback()
		return http.StatusNotFound, err
	}

	userID, ok := userId.(string)
	if !ok {
		tx.Rollback()
		return http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	_, code, err := user.GetUser(userID, tx)
	if err != nil {
		tx.Rollback()
		return code, err
	}

	chanExist, _ := channel.CheckChannelExists(tx, channelID)
	dmChanExist, _ := dmChannel.CheckChannelExists(tx, channelID, userID)
	if !(dmChanExist || chanExist) {
		tx.Rollback()
		return http.StatusNotFound, errors.New("channel does not exist")
	}

    thread.ID = threadID
    err = threadDoc.GetThreadById(threadID)
    if err != nil {
        tx.Rollback()
        return http.StatusNotFound, errors.New("thread not found")
    }

	savedMessageIds := models.SavedMessageIds{
		UserID:   userID,
		OrgID:    threadDoc.OrgansationID,
		ThreadID: threadDoc.ID,
	}

	if exists := savedMessage.SavedThreadMsgExists(tx, savedMessageIds); exists {
		if err := savedMessage.DeleteSavedThreadMsgByMessageID(tx, savedMessageIds); err != nil {
			tx.Rollback()
			return http.StatusBadRequest, err
		}
	}

	pinnedThread := models.PinnedMessage{
		ThreadID:   threadID,
		ChannelsID: channelID,
	}
	if exists := pinnedThread.CheckPinnedReplyExists(tx); exists {
		if err := pinnedThread.DeletePinnedThreadMessageRecord(tx); err != nil {
			tx.Rollback()
			logger.Error("An error occurred while deleting pinned thread message record: %v", err)
			return http.StatusInternalServerError, err
		}
	}

	if _, err := thread.DeleteThread(tx); err != nil {
		tx.Rollback()
		return http.StatusBadRequest, err
	}

	if _, err := thread.DeleteThreadMediaFiles(logger, tx, threadDoc.Media); err != nil {
		tx.Rollback()
		return http.StatusBadRequest, err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to commit transaction: %v", err)
		return http.StatusInternalServerError, err
	}

	notification := models.Notification[models.Deleted]
	notification.SectionType = models.ThreadSection
	notification.ModificationDetails = &models.ModificationDetails{
		ThreadId:  threadID,
		ChannelId: channelID,
	}

	if err := centrifuge.PublishChannel(logger, channelID, notification); err != nil {
		logger.Error("Error Publishing to with destination id: %s error: %v", channelID, err)
		return http.StatusBadRequest, errors.New("failed to publish data")
	}

	return http.StatusOK, nil
}

func UpdateThreadMessage(req models.UpdateThreadMessage, db *gorm.DB, c *gin.Context, logger *utility.Logger) (models.ThreadDocument, int, error) {
	var (
		thread     models.Threads
		threadResp models.ThreadDocument
		user       models.User
		dmChannel  models.DmChannels
		channel    models.Channels
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
	dmChanExist, _ := dmChannel.CheckChannelExists(db, req.ChannelId, userID)

	if !(dmChanExist || chanExist) {
		return threadResp, http.StatusNotFound, errors.New("channel does not exist")
	}

	user, err = user.GetUserByID(db, userID)

	if err != nil {
		return threadResp, http.StatusBadRequest, errors.New("failed to get user")
	}

	thread.ID = req.ThreadId

	updateKey := map[string]any{
		"message": req.Message,
		"edited":  true,
	}

	if _, err := thread.UpdateThread(db, updateKey); err != nil {
		return threadResp, http.StatusNotFound, err
	}

	err = threadResp.GetThreadById(thread.ID)

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
	notification.ModificationDetails = &models.ModificationDetails{
		ThreadId:  req.ThreadId,
		ChannelId: req.ChannelId,
	}

	err = centrifuge.PublishChannel(logger, req.ChannelId, notification)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to with destination id: %s error: %v", req.ChannelId, err.Error()))
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
