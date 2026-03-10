package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/gosimple/slug"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/middleware/common"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility/audit_utility"
)

// Inactivity threshold used to determine if a user is currently in an active streak.
const InactivityThreshold = 30 * time.Minute

func GetActivityLength(u models.User) *string {
	if u.LastActivityAt == nil || u.LastActivityStreakStartedAt == nil {
		return nil
	}

	if time.Since(*u.LastActivityAt) > InactivityThreshold {
		return nil
	}

	duration := time.Since(*u.LastActivityStreakStartedAt)
	hours := int(duration.Hours())
	days := hours / 24
	remHours := hours % 24

	var s string
	if days >= 1 {
		if remHours == 0 {
			if days == 1 {
				s = fmt.Sprintf("active for %d day", days)
			} else {
				s = fmt.Sprintf("active for %d days", days)
			}
		} else {
			dayWord := "days"
			if days == 1 {
				dayWord = "day"
			}
			hourWord := "hours"
			if remHours == 1 {
				hourWord = "hour"
			}
			s = fmt.Sprintf("active for %d %s and %d %s", days, dayWord, remHours, hourWord)
		}
	} else {
		if hours <= 0 {
			s = "active for less than 1 hour"
		} else if hours == 1 {
			s = "active for 1 hour"
		} else {
			s = fmt.Sprintf("active for %d hours", hours)
		}
	}

	return &s
}

func GetUser(userID string, db *gorm.DB) (models.User, int, error) {
	var user models.User

	user, err := user.GetUserByID(db, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user, http.StatusNotFound, errors.New("user not found")
		}
		return user, http.StatusBadRequest, err
	}
	return user, http.StatusOK, nil
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

	orgResp, err = orgData.GetUserOrganisations(db, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &orgResp, http.StatusNotFound, errors.New("user not found")
		}
		return &orgResp, http.StatusBadRequest, err
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
		// Capture old values for audit logging
		oldValues := map[string]interface{}{
			"name":       targetUser.Name,
			"first_name": targetUser.Profile.FirstName,
			"last_name":  targetUser.Profile.LastName,
			"phone":      targetUser.Profile.Phone,
		}
		oldValJSON, _ := json.Marshal(oldValues)

		targetUser.Name = userData.UserName
		targetUser.Profile.FirstName = userData.FirstName
		targetUser.Profile.LastName = userData.LastName
		targetUser.Profile.Phone = userData.PhoneNumber

		err = targetUser.Update(db)
		if err != nil {
			return &targetUser, http.StatusInternalServerError, err
		}

		// Create audit log for profile update
		newValues := map[string]interface{}{
			"name":       targetUser.Name,
			"first_name": targetUser.Profile.FirstName,
			"last_name":  targetUser.Profile.LastName,
			"phone":      targetUser.Profile.Phone,
		}
		newValJSON, _ := json.Marshal(newValues)

		if err := audit_utility.CreateAuditLog(
			db,
			currentUserID,
			currentUser.Email,
			"user",
			models.ActionProfileUpdate,
			models.ResourceUser,
			targetUser.ID,
			string(oldValJSON),
			string(newValJSON),
			fmt.Sprintf("User %s updated profile of %s", currentUser.Email, targetUser.Email),
			audit_utility.GetClientIP(c),
			c.GetHeader("User-Agent"),
		); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("failed to create audit log: %v\n", err)
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

	// Create audit log for user reactivation
	userClaims := common.GetAllUserClaims(ctx)
	actorID, _ := userClaims["user_id"].(string)

	var actorEmail string
	if actorID != "" {
		actor, _, err := GetUser(actorID, db)
		if err == nil {
			actorEmail = actor.Email
		}
	}

	if err := audit_utility.CreateAuditLog(
		db,
		actorID,
		actorEmail,
		"user",
		models.ActionUserReactivate,
		models.ResourceUser,
		user.ID,
		"",
		"",
		fmt.Sprintf("User %s reactivated account for %s", actorEmail, user.Email),
		audit_utility.GetClientIP(ctx),
		ctx.GetHeader("User-Agent"),
	); err != nil {
		fmt.Printf("failed to create audit log: %v\n", err)
	}

	return http.StatusOK, nil
}

func DeactiveUser(db *gorm.DB, userID, loggedInUserID string) (int, error) {
	var user models.User

	if userID == loggedInUserID {
		return http.StatusForbidden, errors.New("you cannot deactivate your own account")
	}

	user, err := user.GetUserByID(db, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusNotFound, errors.New("user not found")
		}
		return http.StatusBadRequest, err
	}

	if err := user.DeactivateUser(db, user.ID); err != nil {
		return http.StatusInternalServerError, err
	}

	// Create audit log for user deactivation
	var actorEmail string
	actor, _, err := GetUser(loggedInUserID, db)
	if err == nil {
		actorEmail = actor.Email
	}

	if err := audit_utility.CreateAuditLog(
		db,
		loggedInUserID,
		actorEmail,
		"user",
		models.ActionUserDeactivate,
		models.ResourceUser,
		user.ID,
		"",
		"",
		fmt.Sprintf("User %s deactivated account for %s", actorEmail, user.Email),
		"", // IP address not available in this context
		"", // User agent not available in this context
	); err != nil {
		fmt.Printf("failed to create audit log: %v\n", err)
	}

	return http.StatusOK, nil
}

func SwitchUserOrg(db *gorm.DB, c *gin.Context, req models.SwitchUserOrgReqeust, userId, accessTokenID string) (gin.H, int, error) {
	var (
		user            models.User
		org             models.Organisation
		orgMgt          models.OrgUserManagement
		accessToken     models.AccessToken
		accessTokenData models.AccessToken
		orgRole         models.OrgRole
		// getOrgRole      models.OrgRole
	)

	orgMgt, err := orgMgt.GetByIDs(db, userId, req.CurrentOrg)
	if err != nil {
		return gin.H{}, http.StatusBadRequest, err
	}

	// getOrgRole, err = getOrgRole.GetAOrgRoleByName(db, "Administrator")
	// if err != nil {
	// 	return gin.H{}, http.StatusBadRequest, err
	// }

	// orgMgt.RoleID = getOrgRole.ID
	// orgMgt.Update(db)

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

	err = user.Update(db)
	if err != nil {
		return gin.H{}, http.StatusInternalServerError, err
	}

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
		return gin.H{}, http.StatusInternalServerError, fmt.Errorf("error saving token: %v", err.Error())
	}

	theData := gin.H{
		"organisation":              org,
		"access_token":              token.AccessToken,
		"current_organisation_slug": slug.Make(org.Name),
	}

	return theData, http.StatusOK, nil
}

func GetAUserForMentions(userIDStr, requestingUserID, orgID string, db *gorm.DB) (models.UserMentionResponse, int, error) {
	var (
		userResp models.UserMentionResponse
	)

	var requestingUserOrgMgt models.OrgUserManagement
	_, err := requestingUserOrgMgt.GetByIDs(db, requestingUserID, orgID)
	if err != nil {
		return userResp, http.StatusForbidden, errors.New("requesting user not in current organization")
	}

	var targetUserOrgMgt models.OrgUserManagement
	_, err = targetUserOrgMgt.GetByIDs(db, userIDStr, orgID)
	if err != nil {
		return userResp, http.StatusForbidden, errors.New("target user not in the same organization")
	}

	var targetUser models.User
	targetUser, err = targetUser.GetUserWithProfile(db, userIDStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return userResp, http.StatusNotFound, errors.New("user not found")
		}
		return userResp, http.StatusInternalServerError, err
	}

	userResp = models.UserMentionResponse{
		UserID:           targetUser.ID,
		Username:         targetUser.Profile.UserName,
		FullName:         targetUser.Profile.FullName,
		FirstName:        targetUser.Profile.FirstName,
		LastName:         targetUser.Profile.LastName,
		AvatarURL:        targetUser.Profile.AvatarURL,
		DefaultAvatarURL: avatar.GenerateDefaultAvatarURL(targetUser.ID),
		DisplayName:      targetUser.Profile.DisplayName,
		StatusText:       targetUser.Profile.Text,
		OnlineStatus:     targetUser.Profile.Online,
	}

	return userResp, http.StatusOK, nil
}
