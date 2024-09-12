package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/services/integrations"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GetOrganisationChannelIntegrations(c *gin.Context) {
	channelId := c.Param("channel_id")
	orgId := c.Param("org_id")

	integrations, paginationResponse, err := integrations.GetOrganisationChannelIntegrations(base.Db.Postgresql, channelId, orgId, c)

	if err != nil {
		base.Logger.Error("Failed to get channel integrations")
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to get channel integrations", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Channel integrations retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Channel integrations retrieved successfully", integrations, paginationResponse)
	c.JSON(http.StatusOK, rd)
}
