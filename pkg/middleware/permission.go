package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware/common"
	rd "github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func PermissionMiddleware(db *gorm.DB, rdb *redis.Client, requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {

		userClaims := common.GetAllUserClaims(c)
		userID, _ := userClaims["user_id"].(string)
		tokenOrgID, _ := userClaims["org_id"].(string)

		requestOrgID := c.Param("org_id")
		targetOrgID := requestOrgID
		if targetOrgID == "" {
			targetOrgID = tokenOrgID
		}

		if targetOrgID != "" && userID != "" {
			if deactivated, err := checkUserDeactivated(db, rdb, userID, targetOrgID); err == nil && deactivated {
				r := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "User is deactivated from organisation", "Unauthorized", nil)
				c.AbortWithStatusJSON(http.StatusUnauthorized, r)
				return
			}
		}

		roleID, ok := userClaims["role_id"].(string)

		if !ok || roleID == "" || (requestOrgID != "" && requestOrgID != tokenOrgID) {
			var err error
			roleID, err = resolveUserOrgRole(db, rdb, userID, targetOrgID)
			if err != nil || roleID == "" {
				r := utility.BuildErrorResponse(http.StatusForbidden, "error", "Forbidden", "You are not a member of this organisation", nil)
				c.AbortWithStatusJSON(http.StatusForbidden, r)
				return
			}
		}

		permissionList, err := resolveRolePermissions(db, rdb, roleID)
		if err != nil {
			r := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Internal Server Error", "Failed to load permissions", nil)
			c.AbortWithStatusJSON(http.StatusInternalServerError, r)
			return
		}

		if !models.OrgUserHasPermission(permissionList, requiredPermission) {
			r := utility.BuildErrorResponse(http.StatusForbidden, "error", "Forbidden", "You do not have the required permissions", nil)
			c.AbortWithStatusJSON(http.StatusForbidden, r)
			return
		}

		c.Next()
	}
}

// resolveUserOrgRole returns the role_id for a user in a target organisation,
// using Redis cache ("user_org_role:{userID}:{orgID}") when available and DB on a miss.
func resolveUserOrgRole(db *gorm.DB, rdb *redis.Client, userID, orgID string) (string, error) {
	if userID == "" || orgID == "" {
		return "", errors.New("missing user_id or org_id")
	}

	cacheKey := "user_org_role:" + userID + ":" + orgID
	cached, err := rd.RedisGet(rdb, cacheKey)
	if err == nil && len(cached) > 0 {
		var roleID string
		if err := json.Unmarshal(cached, &roleID); err == nil && roleID != "" {
			return roleID, nil
		}
	}

	var oum models.OrgUserManagement
	membership, err := oum.GetByIDs(db, userID, orgID)
	if err != nil || membership.RoleID == "" {
		return "", err
	}

	_ = rd.RedisSet(rdb, cacheKey, membership.RoleID, 24*time.Hour)

	return membership.RoleID, nil
}

// resolveRolePermissions returns the permissions for a role, using the Redis
// cache when available and populating it on a miss.
func resolveRolePermissions(db *gorm.DB, rdb *redis.Client, roleID string) (models.PermissionList, error) {
	cacheKey := "role_permissions_" + roleID
	cached, err := rd.RedisGet(rdb, cacheKey)
	if err == nil && len(cached) > 0 {
		var pl models.PermissionList
		if err := json.Unmarshal(cached, &pl); err == nil {
			return pl, nil
		}
	}

	var orgRole models.OrgRole
	role, err := orgRole.GetAOrgRoleById(db, roleID)
	if err != nil {
		return models.PermissionList{}, err
	}

	rd.RedisSet(rdb, cacheKey, role.Permissions.PermissionList, 24*time.Hour)
	return role.Permissions.PermissionList, nil
}

// checkUserDeactivated checks if a user is deactivated from an organisation,
// using Redis cache ("user_deactivated:{userID}:{orgID}") when available and DB on a miss.
func checkUserDeactivated(db *gorm.DB, rdb *redis.Client, userID, orgID string) (bool, error) {
	if userID == "" || orgID == "" {
		return false, nil
	}

	cacheKey := "user_deactivated:" + userID + ":" + orgID
	cached, err := rd.RedisGet(rdb, cacheKey)
	if err == nil && len(cached) > 0 {
		var isDeactivated bool
		if err := json.Unmarshal(cached, &isDeactivated); err == nil {
			return isDeactivated, nil
		}
	}

	var oum models.OrgUserManagement
	ids := models.IDS{
		OrganisationID: orgID,
		UserID:         userID,
	}
	isDeactivated := oum.CheckIsUserDeactivated(db, ids)

	_ = rd.RedisSet(rdb, cacheKey, isDeactivated, 24*time.Hour)
	return isDeactivated, nil
}
