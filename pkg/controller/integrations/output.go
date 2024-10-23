package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/services/integrations"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) FetchOutputIntegrations(c *gin.Context) {
	org_id := c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	integrations, err := integrations.GetActiveOutputIntegrations(base.Db.Postgresql, org_id)
	if err != nil {
		base.Logger.Error("Failed to get out putintegrations")
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to get output integrations", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	base.Logger.Info("output integrations retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "output integrations retrieved successfully.", integrations)
	c.JSON(http.StatusOK, rd)
}
