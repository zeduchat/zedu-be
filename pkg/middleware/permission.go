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

func PermissionMiddleware(db *gorm.DB, rdb *redis.Client, requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {

		userClaims := common.GetAllUserClaims(c)
		roleID, ok := userClaims["role_id"].(string)
		if !ok || roleID == "" {
			r := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Unauthorized", "Missing role", nil)
			c.AbortWithStatusJSON(http.StatusUnauthorized, r)
			return
		}

		userID, _ := userClaims["user_id"].(string)
		tokenOrgID, _ := userClaims["org_id"].(string)

		// If the route targets a specific org, verify the token's org matches.
		// When they differ, resolve the user's actual role in the target org.
		requestOrgID := c.Param("org_id")
		if requestOrgID != "" && requestOrgID != tokenOrgID {
			var oum models.OrgUserManagement
			membership, err := oum.GetByIDs(db, userID, requestOrgID)
			if err != nil {
				r := utility.BuildErrorResponse(http.StatusForbidden, "error", "Forbidden", "You are not a member of this organisation", nil)
				c.AbortWithStatusJSON(http.StatusForbidden, r)
				return
			}
			roleID = membership.RoleID
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
