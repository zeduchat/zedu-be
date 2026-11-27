package agents

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware/common"
	"github.com/hngprojects/telex_be/services/organisation"
	"github.com/hngprojects/telex_be/utility"
)

// RequireAgentManager must run after Authorize. It reads :org_id from the path
// and aborts with 403 unless the caller owns that organisation or holds
// can_manage_agents there through an active membership. Non-members, unknown
// orgs and deactivated members all get the same 403.
func (base *Controller) RequireAgentManager(c *gin.Context) {
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

	if !organisation.UserCanOrOwner(base.Db.Postgresql, userID, orgID, models.PermManageAgents) {
		c.AbortWithStatusJSON(http.StatusForbidden, utility.BuildErrorResponse(http.StatusForbidden, "error", "you do not have permission to manage agents in this organisation", "Forbidden", nil))
		return
	}

	c.Next()
}
