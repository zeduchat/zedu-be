package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	admin "github.com/hngprojects/telex_be/services/admin"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) InviteLeaderboard(c *gin.Context) {
	orgID := c.Query("org_id")
	var orgPtr *string
	if orgID != "" {
		if _, err := uuid.Parse(orgID); err != nil {
			base.Logger.Error("invalid org_id format", err)
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid org_id format", "invalid org_id format", nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		orgPtr = &orgID
	}

	limit := 10
	if c.Query("limit") != "" {
		l, err := strconv.Atoi(c.Query("limit"))
		if err != nil || l <= 0 {
			if err != nil {
				base.Logger.Error("invalid limit - parse error", err)
			} else {
				base.Logger.Error("invalid limit - non-positive", fmt.Errorf("limit must be a positive integer: %d", l))
			}
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid limit", "limit must be a positive integer", nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		if l > 100 {
			base.Logger.Error("limit exceeds maximum allowed", fmt.Errorf("limit cannot exceed 100: %d", l))
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "limit cannot exceed 100", "limit cannot exceed 100", nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		limit = l
	}

	users, code, err := admin.ListUsersByInvites(base.Db.Postgresql, orgPtr, limit)
	if err != nil {
		base.Logger.Error("failed to build invite leaderboard", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("invite leaderboard retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "invite leaderboard retrieved successfully", users)
	c.JSON(http.StatusOK, rd)
}

// GetInvitationDashboard returns comprehensive invitation dashboard data
func (base *Controller) GetInvitationDashboard(c *gin.Context) {
	includeStats := c.DefaultQuery("include_stats", "true") != "false"

	topLimit := 5
	if c.Query("top_limit") != "" {
		l, err := strconv.Atoi(c.Query("top_limit"))
		if err == nil && l > 0 && l <= 20 {
			topLimit = l
		}
	}

	filter := admin.InvitationFilter{
		Search:       c.Query("search"),
		Status:       c.Query("status"),
		IncludeStats: includeStats,
		TopLimit:     topLimit,
	}

	response, paginationResponse, code, err := admin.GetInvitationDashboard(base.Db.Postgresql, c, filter)
	if err != nil {
		base.Logger.Error("failed to get invitation dashboard", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("invitation dashboard retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "invitation dashboard retrieved successfully", response, paginationResponse)
	c.JSON(http.StatusOK, rd)
}
