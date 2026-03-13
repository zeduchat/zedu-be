package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	admin "github.com/hngprojects/telex_be/services/admin"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GetAppActivity(c *gin.Context) {
	includeStats := c.DefaultQuery("include_stats", "true") != "false"

	filter := admin.ActivityFilter{
		Search:       c.Query("search"),
		Role:         c.Query("role"),
		Status:       c.Query("status"),
		Action:       c.Query("action"),
		Duration:     c.Query("duration"),
		StartDate:    c.Query("start_date"),
		EndDate:      c.Query("end_date"),
		IncludeStats: includeStats,
	}

	response, paginationResponse, code, err := admin.ListAppActivity(base.Db.Postgresql, c, filter)
	if err != nil {
		base.Logger.Error("failed to list app activity", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("app activity retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "app activity retrieved successfully", response, paginationResponse)
	c.JSON(http.StatusOK, rd)
}
