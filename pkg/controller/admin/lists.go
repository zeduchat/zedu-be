package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	admin "github.com/hngprojects/telex_be/services/admin"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) ListUsers(c *gin.Context) {
	includeStats := c.DefaultQuery("include_stats", "true") != "false"

	filter := admin.UserFilter{
		Search:       c.Query("search"),
		UserType:     c.Query("user_type"),
		Duration:     c.Query("duration"),
		StartDate:    c.Query("start_date"),
		EndDate:      c.Query("end_date"),
		IncludeStats: includeStats,
	}

	response, paginationResponse, code, err := admin.ListUsers(base.Db.Postgresql, c, filter)
	if err != nil {
		base.Logger.Error("failed to list users", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("users retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "users retrieved successfully", response, paginationResponse)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetPlatformCreditsSummary(c *gin.Context) {
	metrics, err := models.GetPlatformCreditSummary(base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to fetch platform credit summary", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to fetch platform credit summary", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Platform credit summary retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Platform credit summary retrieved successfully", metrics)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetFreeVsPaidUserStats(c *gin.Context) {
	duration := c.Query("duration")

	resp, code, err := admin.GetFreeVsPaidUserStats(base.Db.Postgresql, duration)
	if err != nil {
		base.Logger.Error("Failed to fetch user stats", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to fetch user stats", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("User stats retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "User stats retrieved successfully", resp)
	c.JSON(http.StatusOK, rd)
}
