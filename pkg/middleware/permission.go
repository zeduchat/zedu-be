package middleware

import (
	"encoding/json"
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

// PermissionMiddleware allows the request only when the caller's current role
// in the target organisation grants requiredPermission. The membership row is
// read from the database on every request: the token's role_id claim is
// minted at login and goes stale when the member is demoted, so it is never
// used here. The request fails closed when there is no user, no organisation,
// no membership or no role.
func PermissionMiddleware(db *gorm.DB, rdb *redis.Client, requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {

		userClaims := common.GetAllUserClaims(c)
		userID, _ := userClaims["user_id"].(string)
		tokenOrgID, _ := userClaims["org_id"].(string)

		targetOrgID := c.Param("org_id")
		if targetOrgID == "" {
			targetOrgID = tokenOrgID
		}

		if userID == "" || targetOrgID == "" {
			r := utility.BuildErrorResponse(http.StatusForbidden, "error", "Forbidden", "You are not a member of this organisation", nil)
			c.AbortWithStatusJSON(http.StatusForbidden, r)
			return
		}

		var oum models.OrgUserManagement
		membership, err := oum.GetByIDs(db, userID, targetOrgID)
		if err != nil || membership.RoleID == "" {
			r := utility.BuildErrorResponse(http.StatusForbidden, "error", "Forbidden", "You are not a member of this organisation", nil)
			c.AbortWithStatusJSON(http.StatusForbidden, r)
			return
		}

		if membership.IsDeactivated {
			r := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "User is deactivated from organisation", "Unauthorized", nil)
			c.AbortWithStatusJSON(http.StatusUnauthorized, r)
			return
		}

		permissionList, err := resolveRolePermissions(db, rdb, membership.RoleID)
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
