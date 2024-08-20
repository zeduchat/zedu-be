package user

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/middleware/common"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

func GetAUserLoginActivity(userIDStr string, db *gorm.DB, c *gin.Context) (*[]models.LoginActivity, *postgresql.PaginationResponse, int, error) {
	var (
		loginData models.LoginActivity
		loginResp []models.LoginActivity
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return nil, nil, http.StatusNotFound, err
	}

	userID, ok := userId.(string)
	if !ok {
		return nil, nil, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	user, code, err := GetUser(userID, db)
	if err != nil {
		return nil, nil, code, err
	}

	isSuperAdmin := user.CheckUserIsAdmin(db)
	loginResp, paginationResponse, err := loginData.GetLoginActivityByIDsAdmin(db, c, userIDStr, userID, isSuperAdmin)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &loginResp, nil, http.StatusNoContent, nil
		}
		return &loginResp, nil, http.StatusBadRequest, err

	}

	return &loginResp, &paginationResponse, http.StatusOK, nil
}

func GetAUserSessions(userIDStr string, db *gorm.DB, c *gin.Context) (*[]models.AccessToken, *postgresql.PaginationResponse, int, error) {
	var (
		accessData models.AccessToken
		accessResp []models.AccessToken
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return nil, nil, http.StatusNotFound, err
	}

	userID, ok := userId.(string)
	if !ok {
		return nil, nil, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	user, code, err := GetUser(userID, db)
	if err != nil {
		return nil, nil, code, err
	}

	isSuperAdmin := user.CheckUserIsAdmin(db)
	accessResp, paginationResponse, err := accessData.FetchAllUserSessions(db, c, userIDStr, userID, isSuperAdmin)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &accessResp, nil, http.StatusNoContent, nil
		}
		return &accessResp, nil, http.StatusBadRequest, err

	}

	return &accessResp, &paginationResponse, http.StatusOK, nil
}

func RevokeUserAccessToken(userData models.TerminateSessionRequest, db *gorm.DB, c *gin.Context) (int, error) {
	var (
		currentUser models.User
		accessToken models.AccessToken
	)
	userClaims := common.GetAllUserClaims(c, "userClaims")

	currentUserID, ok := userClaims["user_id"].(string)
	if !ok {
		return http.StatusBadGateway, fmt.Errorf("error getting user id")
	}

	currentUser, code, err := GetUser(currentUserID, db)
	if err != nil {
		return code, err
	}

	_, code, err = GetUser(*userData.UserID, db)
	if err != nil {
		return code, err
	}

	isSuperAdmin := currentUser.CheckUserIsAdmin(db)
	if currentUserID == *userData.UserID || isSuperAdmin {
		if userData.AccessToken != nil {

			accessToken, err := accessToken.GetAccessTokenByID(db, *userData.AccessToken)
			if err != nil {
				return http.StatusNotFound, err
			}

			if err := accessToken.RevokeAccessToken(db); err != nil {
				return http.StatusBadRequest, fmt.Errorf("error revoking user session: %v", err)
			}

		} else if userData.GlobalTermination {

			if isSuperAdmin {
				var access models.AccessToken
				if err := access.RevokeAllTokens(db); err != nil {
					return http.StatusBadRequest, err
				}
			} else {
				return http.StatusForbidden, errors.New("only an admin can revoke all sessions")
			}
		} else {

			accessUUID, ok := userClaims["access_uuid"].(string)
			if !ok {
				return http.StatusBadGateway, fmt.Errorf("error getting session id: %v", err)
			}
			accessToken, err := accessToken.GetAccessTokenByID(db, accessUUID)
			if err != nil {
				return http.StatusNotFound, err
			}
			if err := accessToken.RevokeAccessToken(db); err != nil {
				return http.StatusBadRequest, fmt.Errorf("error revoking user session: %v", err)
			}
		}
	} else {
		return http.StatusForbidden, errors.New("user does not have permission to update this user")
	}

	return http.StatusOK, nil
}
