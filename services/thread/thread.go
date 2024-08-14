package thread

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/user"
	"gorm.io/gorm"
)

func GetAllUserOrgThreads(orgID string, db *gorm.DB, c *gin.Context) (*[]models.Threads, *postgresql.PaginationResponse, int, error) {
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
		return &accessResp, nil, http.StatusBadRequest, err

	}

	return &accessResp, &paginationResponse, http.StatusOK, nil
}

func GetAllChannelThreads(channelID string, db *gorm.DB, c *gin.Context) (*[]models.Threads, *postgresql.PaginationResponse, int, error) {
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
			return &accessResp, nil, http.StatusNoContent, nil
		}
		return &accessResp, nil, http.StatusBadRequest, err

	}

	return &accessResp, &paginationResponse, http.StatusOK, nil
}

func GetUserSingleThreads(threadID string, db *gorm.DB, c *gin.Context) (*models.Threads, int, error) {
	var (
		accessData models.Threads
		accessResp *models.Threads
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return nil, http.StatusNotFound, err
	}

	userID, ok := userId.(string)
	if !ok {
		return nil, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	_, code, err := user.GetUser(userID, db)
	if err != nil {
		return nil, code, err
	}

	accessResp, err = accessData.GetSingleThreadWithReplies(db, threadID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return accessResp, http.StatusNoContent, nil
		}
		return accessResp, http.StatusBadRequest, err

	}

	return accessResp, http.StatusOK, nil
}

func UpdateAThread(req models.UpdateThreadStatus, threadID string, db *gorm.DB, c *gin.Context) (int, error) {
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

	threadData, err := thread.GetThreadByIds(db, threadID, userID)
	if err != nil {
		return http.StatusBadRequest, err
	}

	threadData.ThreadStatus = req.ThreadStatus

	if _, err := threadData.UpdateThread(db); err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}
