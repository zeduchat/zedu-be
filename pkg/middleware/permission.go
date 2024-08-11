package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func PermissionMiddleware(requiredPermissions []string, role models.OrgRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !models.OrgUserHasAnyPermission(role, requiredPermissions...) {
			r := utility.BuildErrorResponse(http.StatusForbidden, "error",
				"You do not have the required permissions", "Forbidden", nil)
			c.AbortWithStatusJSON(http.StatusForbidden, r)
			return
		}
		c.Next()
	}
}
