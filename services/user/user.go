package user

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/middleware/common"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

func GetUser(userIDStr string, db *gorm.DB) (models.User, int, error) {
	var userResp models.User

	userResp, err := userResp.GetUserByID(db, userIDStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return userResp, http.StatusNotFound, errors.New("user not found")
		}
		return userResp, http.StatusBadRequest, err
	}
	return userResp, http.StatusOK, nil
}

func GetUserByEmail(email string, db *gorm.DB) (models.User, error) {
	var user models.User

	user, err := user.GetUserByEmail(db, email)
	if err != nil {
		return user, err
	}
	return user, nil
}

func GetAUser(userIDStr string, db *gorm.DB, c *gin.Context) (*models.User, int, error) {
	var userResp models.User

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return nil, http.StatusNotFound, err
	}

	userID, ok := userId.(string)
	if !ok {
		return nil, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	user, code, err := GetUser(userID, db)
	if err != nil {
		return nil, code, err
	}

	isSuperAdmin := user.CheckUserIsAdmin(db)
	if isSuperAdmin {
		userResp, err = userResp.GetUserByID(db, userIDStr)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &userResp, http.StatusNotFound, errors.New("user not found")
			}
			return &userResp, http.StatusBadRequest, err
		}
	} else {
		userResp, err = userResp.GetUserByIDsAdmin(db, userIDStr, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &userResp, http.StatusNotFound, errors.New("user not found")
			}
			return &userResp, http.StatusBadRequest, err
		}
	}

	return &userResp, http.StatusOK, nil
}

func GetOrganisationDetails(db *gorm.DB, c *gin.Context, org_id string) (*models.Organisation, int, error) {
	var orgData models.Organisation
	orgResp, err := orgData.GetOrganisationDetails(db, org_id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &orgResp, http.StatusNotFound, errors.New("organisation not found")
		}
		return &orgResp, http.StatusBadRequest, err
	}
	return &orgResp, http.StatusOK, nil
}

func GetAUserOrganisation(db *gorm.DB, c *gin.Context) (*[]models.Organisation, int, error) {
	var (
		orgData models.Organisation
		orgResp []models.Organisation
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return nil, http.StatusNotFound, err
	}

	userID, ok := userId.(string)
	if !ok {
		return nil, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	user, code, err := GetUser(userID, db)
	if err != nil {
		return nil, code, err
	}

	isSuperAdmin := user.CheckUserIsAdmin(db)
	if isSuperAdmin {
		orgResp, err = orgData.GetOrganisationsByUserID(db, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &orgResp, http.StatusNotFound, errors.New("user not found")
			}
			return &orgResp, http.StatusBadRequest, err
		}
	} else {
		orgResp, err = orgData.GetOrganisationsByUserIDs(db, userID, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &orgResp, http.StatusNotFound, errors.New("user not found")
			}
			return &orgResp, http.StatusBadRequest, err
		}
	}

	return &orgResp, http.StatusOK, nil
}

func DeleteAUser(userIDStr string, db *gorm.DB, c *gin.Context) (int, error) {
	var (
		currentUser models.User
		targetUser  models.User
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return http.StatusNotFound, err
	}

	currentUserID, ok := userId.(string)
	if !ok {
		return http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	currentUser, code, err := GetUser(currentUserID, db)
	if err != nil {
		return code, err
	}

	targetUser, code, err = GetUser(userIDStr, db)
	if err != nil {
		return code, err
	}

	isSuperAdmin := currentUser.CheckUserIsAdmin(db)
	if isSuperAdmin || currentUserID == userIDStr {

		if err := targetUser.DeleteAUser(db); err != nil {
			return http.StatusInternalServerError, err
		}
	} else {
		return http.StatusForbidden, errors.New("user does not have permission to delete this user")
	}

	return http.StatusOK, nil
}

func UpdateAUser(userData models.UpdateUserRequestModel, userIDStr string, db *gorm.DB, c *gin.Context) (*models.User, int, error) {
	var (
		currentUser models.User
		targetUser  models.User
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return &targetUser, http.StatusNotFound, err
	}

	currentUserID, ok := userId.(string)
	if !ok {
		return &targetUser, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	currentUser, code, err := GetUser(currentUserID, db)
	if err != nil {
		return &targetUser, code, err
	}

	targetUser, code, err = GetUser(userIDStr, db)
	if err != nil {
		return &targetUser, code, err
	}

	isSuperAdmin := currentUser.CheckUserIsAdmin(db)
	if isSuperAdmin || currentUserID == userIDStr {

		targetUser.Name = userData.UserName
		targetUser.Profile.FirstName = userData.FirstName
		targetUser.Profile.LastName = userData.LastName
		targetUser.Profile.Phone = userData.PhoneNumber

		err = targetUser.Update(db)
		if err != nil {
			return &targetUser, http.StatusInternalServerError, err
		}

	} else {
		return &targetUser, http.StatusForbidden, errors.New("user does not have permission to update this user")
	}

	return &targetUser, http.StatusOK, nil
}

func GetAllUsers(c *gin.Context, db *gorm.DB) ([]models.User, *postgresql.PaginationResponse, int, error) {

	var users []models.User
	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(db, "created_at", "desc", pagination, &users, "deleted_at IS NULL")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return users, nil, http.StatusNoContent, nil
		}
		return users, nil, http.StatusBadRequest, err

	}

	return users, &paginationResponse, http.StatusOK, nil

}

func ActivateUser(userIDStr string, db *gorm.DB, ctx *gin.Context) (int, error) {
	var user models.User

	user, err := user.GetUserByID(db, userIDStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusNotFound, errors.New("user not found")
		}
		return http.StatusBadRequest, err
	}

	if err := user.ActivateUser(db, user.ID); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func DeactiveUser(userIDStr string, db *gorm.DB, ctx *gin.Context) (int, error) {
	var user models.User

	user, err := user.GetUserByID(db, userIDStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusNotFound, errors.New("user not found")
		}
		return http.StatusBadRequest, err
	}

	if err := user.DeactivateUser(db, user.ID); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func SwitchUserOrg(req models.SwitchUserOrgReqeust, userId string,
	db *gorm.DB, c *gin.Context) (gin.H, int, error) {
	var (
		user            models.User
		org             models.Organisation
		orgMgt          models.OrgUserManagement
		accessToken     models.AccessToken
		accessTokenData models.AccessToken
		orgRole         models.OrgRole
	)

	userClaims := common.GetAllUserClaims(c)
	accessTokenID, ok := userClaims["access_uuid"].(string)
	if !ok {
		return gin.H{}, http.StatusBadGateway, fmt.Errorf("error getting access token id")
	}

	orgMgt, err := orgMgt.GetByIDs(db, userId, req.CurrentOrg)
	if err != nil {
		return gin.H{}, http.StatusBadRequest, err
	}

	orgRole, err = orgRole.GetAOrgRoleById(db, orgMgt.RoleID)
	if err != nil {
		return gin.H{}, http.StatusBadRequest, err
	}

	org, err = org.GetOrgByID(db, req.CurrentOrg)
	if err != nil {
		return gin.H{}, http.StatusBadRequest, err
	}

	user, err = user.GetUserByID(db, userId)
	if err != nil {
		return gin.H{}, http.StatusInternalServerError, err

	}

	exist, err := org.CheckUserIsMemberOfOrg(userId, req.CurrentOrg, db)
	if !exist && err != nil {
		return gin.H{}, http.StatusBadRequest, err
	}

	user.CurrentOrg, err = uuid.FromString(req.CurrentOrg)
	if err != nil {
		return gin.H{}, http.StatusInternalServerError, err
	}

	user.OrgRoleID = &orgMgt.RoleID
	user.OrgRole = orgRole

	accessTokenData, err = accessToken.GetAccessTokenByID(db, accessTokenID)
	if err != nil {
		return gin.H{}, http.StatusNotFound, err
	}

	if err := accessTokenData.RevokeAccessTokenDelete(db); err != nil {
		return gin.H{}, http.StatusBadRequest, fmt.Errorf("error revoking user session: %v", err)
	}

	token, err := middleware.CreateToken(user, c)
	if err != nil {
		return gin.H{}, http.StatusBadRequest, err
	}

	tokens := map[string]string{
		"access_token": token.AccessToken,
		"exp":          strconv.Itoa(int(token.ExpiresAt.Unix())),
	}

	access_token := models.AccessToken{ID: token.AccessUuid, OwnerID: user.ID}
	err = access_token.CreateAccessToken(db, tokens)
	if err != nil {
		return gin.H{}, http.StatusInternalServerError, fmt.Errorf("error saving token: " + err.Error())
	}

	err = user.Update(db)
	if err != nil {
		return gin.H{}, http.StatusInternalServerError, err
	}

	theData := gin.H{
		"organisation": org,
		"access_token": token.AccessToken,
	}

	return theData, http.StatusOK, nil

}
