package thread

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/services/user"
)

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

func GetAllChannelThreads(channelID string, db *gorm.DB, c *gin.Context) ([]models.Threads, *elastic.PaginationResponse, int, error) {
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

	accessResp, paginationResponse, err := accessData.GetAllThreadsByChannelID(c, db, userID, channelID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return accessResp, nil, http.StatusNoContent, nil
		}
		return accessResp, nil, http.StatusInternalServerError, err

	}

	return accessResp, paginationResponse, http.StatusOK, nil
}

func GetChannelThreads(channelID string, db *gorm.DB, c *gin.Context) ([]models.Threads, *elastic.PaginationResponse, int, error) {
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
