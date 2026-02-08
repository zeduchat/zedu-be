package admin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	admin "github.com/hngprojects/telex_be/services/admin"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GetUserGrowth(c *gin.Context) {
	params := admin.UserGrowthParams{
		Preset:   c.Query("preset"),
		GroupBy:  c.Query("group_by"),
		Timezone: c.Query("timezone"),
	}

	from, to, err := utility.ParseDateRange(c.Query("from"), c.Query("to"))
	if err != nil {
		base.Logger.Error("Invalid date range", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	params.From = from
	params.To = to

	if params.Preset == "" && (params.From == nil || params.To == nil) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Either preset or both from/to dates must be provided", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if params.Preset != "" && (params.From != nil || params.To != nil) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Cannot use both preset and custom date range", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if params.Preset != "" {
		validPresets := map[string]bool{
			"today":        true,
			"yesterday":    true,
			"last_7_days":  true,
			"last_30_days": true,
			"this_month":   true,
			"this_year":    true,
		}
		if !validPresets[params.Preset] {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid preset. Valid values: today, yesterday, last_7_days, last_30_days, this_month, this_year", nil, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
	}

	if params.GroupBy != "" {
		validGroupBy := map[string]bool{
			"day":   true,
			"week":  true,
			"month": true,
		}
		if !validGroupBy[params.GroupBy] {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid group_by. Valid values: day, week, month", nil, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
	}

	if params.From != nil && params.To != nil {
		if params.To.Before(*params.From) {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "to date must be after from date", nil, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}

		if params.To.Sub(*params.From) > 730*24*time.Hour {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Date range cannot exceed 2 years", nil, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
	}

	result, err := admin.GetUserGrowthMetrics(base.Db.Postgresql, params)
	if err != nil {
		base.Logger.Error("Failed to retrieve user growth data", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to retrieve user growth data", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("User growth data retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "User growth data retrieved successfully", result)
	c.JSON(http.StatusOK, rd)
}
