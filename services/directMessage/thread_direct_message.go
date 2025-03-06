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
	"github.com/hngprojects/telex_be/services/user"
	"github.com/hngprojects/telex_be/utility"
)

func SaveThreadDmMessage(req models.CreateThreadMsgReq, db *storage.Database, logger *utility.Logger) (*models.ThreadDocument, error) {

	var (
		profile models.Profile
		user    models.User
		channel models.DmChannels
	)

	err := profile.GetProfileByUserId(db.Postgresql, req.UserId)

	if err != nil {
		return nil, errors.New("failed to get user profile")
	}

	user, err = user.GetUserByID(db.Postgresql, req.UserId)

	if err != nil {
		return nil, errors.New("failed to get user")
	}

	ch, err := channel.CheckChannelExists(db.Postgresql, req.ChannelsID)

	if !ch || err != nil {
		return nil, errors.New("channel does not exist")
	}

	threadDoc := models.ThreadDocument{
		ID:            req.ThreadId,
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
		Status:        "error",
	}

	err = threadDoc.CreateThread(db, logger)
	if err != nil {
		return nil, err
	}

	feed := models.FeedMessageRequest{
		ChannelID: req.ChannelsID,
		UserName:  profile.UserName,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		AvatarURL: profile.AvatarURL,
		Type:      "message",
		Content:   req.Content,
		ThreadId:  req.ThreadId,
		Email:     user.Email,
		FullName:  profile.FullName,
		UserId:    req.UserId,
	}

	err = centrifuge.BroadcastChannel(logger, req.ChannelsID, feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Broadcasting to channelid: %s, error: %v", req.ChannelsID, err.Error()))
		return nil, errors.New("failed to broadcast webhook data: " + err.Error())
	}

	err = centrifuge.BroadcastChannel(logger, channel.PaticipantId, feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Broadcasting to channelid: %s, with orgid: %s error: %v", req.ChannelsID, req.OrgId, err.Error()))
		return nil, errors.New("failed to broadcast webhook data: " + err.Error())
	}

	return &threadDoc, nil
}

// main channel thread
func CreateThreadDmMessage(req models.CreateThreadMsgReq, db *storage.Database, logger *utility.Logger) (*models.ThreadDocument, error) {

	// Provision for bot dms later

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

func DeleteAThreadDm(threadID, channelID string, db *gorm.DB, c *gin.Context) (int, error) {
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

	if _, err := thread.DeleteThread(db); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
