package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware/common"
	"github.com/hngprojects/telex_be/utility"
)

// RequireOrgMember must run after Authorize. It reads :org_id from the path and
// the user_id claim that Authorize stores under "userClaims", and aborts with
// 403 unless the caller is a member of that organisation. A missing org gets
// the same 403, so the response doesn't reveal whether the org exists.
func RequireOrgMember(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID := c.Param("org_id")
		if !utility.IsValidUUID(orgID) {
			c.AbortWithStatusJSON(http.StatusBadRequest, utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil))
			return
		}

		userID, _ := common.GetAllUserClaims(c)["user_id"].(string)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Token is invalid!", "Unauthorized", nil))
			return
		}

		var org models.Organisation
		isMember, err := org.CheckUserIsMemberOfOrg(userID, orgID, db)
		if err != nil || !isMember {
			c.AbortWithStatusJSON(http.StatusForbidden, utility.BuildErrorResponse(http.StatusForbidden, "error", "you are not a member of this organisation", "Forbidden", nil))
			return
		}

		c.Next()
	}
}
