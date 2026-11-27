package group

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware/common"
	"github.com/hngprojects/telex_be/services/organisation"
	"github.com/hngprojects/telex_be/utility"
)

// RequireGroupManager must run after Authorize and RequireOrgMember. It aborts
// with 403 unless the caller owns :org_id or holds can_manage_channels there.
// Groups have no permission of their own, so they are managed like channels.
func (base *Controller) RequireGroupManager(c *gin.Context) {
	userID, _ := common.GetAllUserClaims(c)["user_id"].(string)
	if userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Token is invalid!", "Unauthorized", nil))
		return
	}

	if !organisation.UserCanOrOwner(base.Db.Postgresql, userID, c.Param("org_id"), models.PermManageChannels) {
		c.AbortWithStatusJSON(http.StatusForbidden, utility.BuildErrorResponse(http.StatusForbidden, "error", "you do not have permission to manage groups in this organisation", "Forbidden", nil))
		return
	}

	c.Next()
}
