package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"

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
		fmt.Println(userClaims)
		roleID, ok := userClaims["role_id"].(string)

		if !ok {
			r := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Unauthorized", "Missing role", nil)
			c.AbortWithStatusJSON(http.StatusUnauthorized, r)
			return
		}

		cacheKey := "role_permissions_" + roleID
		cachedPermissions, err := rd.RedisGet(rdb, cacheKey)
		if err == redis.Nil || len(cachedPermissions) == 0 {
			var orgRole models.OrgRole
			permissions, err := orgRole.GetAOrgRoleById(db, roleID)
			if err != nil {
				r := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Internal Server Error", "Failed to load permissions", nil)
				c.AbortWithStatusJSON(http.StatusInternalServerError, r)
				return
			}

			rd.RedisSet(rdb, cacheKey, permissions.Permissions.PermissionList)
		} else {
			var permissionList models.PermissionList
			err = json.Unmarshal(cachedPermissions, &permissionList)
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
		}

		c.Next()
	}
}
