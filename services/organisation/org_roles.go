package organisation

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	rd "github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateOrgRoles(req models.OrgRole, orgID string, db *gorm.DB, c *gin.Context) (gin.H, int, error) {
	var (
		org  models.Organisation
		user models.User
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return nil, http.StatusNotFound, err
	}

	currentUserID, ok := userId.(string)
	if !ok {
		return nil, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	currentUser, err := user.GetUserByID(db, currentUserID, orgID)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	orgData, err := org.CheckOrgExists(orgID, db)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return gin.H{}, http.StatusNotFound, errors.New("organisation not found")
		}
		return gin.H{}, http.StatusBadRequest, err
	}

	if !userCanOrOwner(db, currentUser.ID, orgData.ID, models.PermCreateCustomRole) {
		return nil, http.StatusForbidden, errors.New("you do not have permission to create roles")
	}

	req.ID = utility.GenerateUUID()
	req.OrganisationID = &orgData.ID
	req.IsDefault = false

	if err := req.CreateOrgRole(db); err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			return gin.H{}, http.StatusConflict, errors.New("role name already exists")
		}
		return gin.H{}, http.StatusBadRequest, err
	}

	theResp := gin.H{
		"id":          req.ID,
		"name":        req.Name,
		"description": req.Description,
		"permissions": req.Permissions.PermissionList.ToMap(),
		"message":     "Role created successfully",
	}

	return theResp, http.StatusCreated, nil
}

func GetOrgRoles(db *gorm.DB, orgID string, c *gin.Context) ([]models.OrgRole, int, error) {
	var (
		orgMgt    models.OrgUserManagement
		role      models.OrgRole
		rolesData []models.OrgRole
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return nil, http.StatusNotFound, err
	}

	currentUserID, ok := userId.(string)
	if !ok {
		return nil, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	orgMgt, err = orgMgt.GetByIDs(db, currentUserID, orgID)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	rolesData, err = role.GetOrgRoles(db, orgID)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	return rolesData, http.StatusOK, nil

}

func GetAOrgRole(db *gorm.DB, orgID, roleID string, c *gin.Context) (*models.OrgRole, int, error) {
	var (
		org       models.Organisation
		role      models.OrgRole
		rolesData models.OrgRole
		user      models.User
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return nil, http.StatusNotFound, err
	}

	currentUserID, ok := userId.(string)
	if !ok {
		return nil, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	currentUser, err := user.GetUserByID(db, currentUserID, orgID)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	orgData, err := org.CheckOrgExists(orgID, db)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusNotFound, errors.New("organisation not found")
		}
		return nil, http.StatusBadRequest, err
	}

	if !userCanOrOwner(db, currentUser.ID, orgData.ID, models.PermCreateCustomRole) {
		return nil, http.StatusForbidden, errors.New("you do not have permission to view this role")
	}

	rolesData, err = role.GetAOrgRole(db, orgID, roleID)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	return &rolesData, http.StatusOK, nil

}

func DeleteOrgRole(db *gorm.DB, rdb *redis.Client, orgID, roleID string, c *gin.Context) (int, error) {
	var (
		org      models.Organisation
		role     models.OrgRole
		roleData models.OrgRole
		user     models.User
		orgMgt   models.OrgUserManagement
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return http.StatusNotFound, err
	}

	currentUserID, ok := userId.(string)
	if !ok {
		return http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	currentUser, err := user.GetUserByID(db, currentUserID, orgID)
	if err != nil {
		return http.StatusBadRequest, err
	}

	orgData, err := org.CheckOrgExists(orgID, db)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusNotFound, errors.New("organisation not found")
		}
		return http.StatusBadRequest, err
	}

	if !userCanOrOwner(db, currentUser.ID, orgData.ID, models.PermCreateCustomRole) {
		return http.StatusForbidden, errors.New("you do not have permission to delete roles")
	}

	roleData, err = role.GetAOrgRole(db, orgID, roleID)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if roleData.IsDefault {
		return http.StatusForbidden, errors.New("can't delete default role")
	}

	// Collect affected user IDs before reassignment so we can revoke their sessions
	var affectedUserIDs []string
	db.Model(&models.OrgUserManagement{}).
		Where("organisation_id = ? AND role_id = ?", orgID, roleID).
		Pluck("user_id", &affectedUserIDs)

	if err := orgMgt.UpdateAllOrgUsersWithNewRole(db, orgID, roleID); err != nil {
		return http.StatusBadRequest, err
	}

	if err := roleData.DeleteOrgRole(db); err != nil {
		return http.StatusBadRequest, err
	}

	// Revoke sessions for affected users so they re-authenticate with the new role
	var accessToken models.AccessToken
	_ = accessToken.RevokeTokensByUserIDs(db, affectedUserIDs)

	// Invalidate the deleted role's permission cache
	rd.RedisDelete(rdb, "role_permissions_"+roleID)

	return http.StatusOK, nil
}
