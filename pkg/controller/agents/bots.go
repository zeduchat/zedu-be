package agents

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/services/agents"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) FetchOrganisationBots(c *gin.Context) {
	var org_id string = c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	response, paginationResponse, code, err := agents.FetchOrganisationBots(base.Db.Postgresql, base.Logger, org_id, c, base.ExtReq, base.Db.Redis)
	if err != nil {
		base.Logger.Error("failed to fetch bots in organisation", err)
		rd := utility.BuildErrorResponse(code, "error", "failed to fetch bots in organisation", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(response),
	}

	base.Logger.Info("Bots in organisation fetched successfully")
	rd := utility.BuildSuccessResponse(code, "Bots in organisation fetched successfully", response, paginationData)
	c.JSON(code, rd)
}
