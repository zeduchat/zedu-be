package plan

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func CheckUserOrgPlanThreshold(c *gin.Context, logger *utility.Logger, db *gorm.DB, organisationID string) bool {
	planData, exists := c.Get("currentPlan")
	if !exists {
		logger.Error("unable to get organization plan data")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get organization plan data", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return false
	}

	plan := planData.(models.Plan)

	// Count number of users in the organization
	var userCount int64
	err := db.Model(&models.OrgUserManagement{}).
		Where("organisation_id = ?", organisationID).
		Count(&userCount).Error

	if err != nil {
		logger.Error("Failed to count organization users")
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Internal Server Error", "Failed to count organization users", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return false
	}

	// Check if user count exceeds plan limit (plan maximum users)
	if userCount >= int64(plan.MaxUsers) {
		logger.Error("Maximum number of users for org plan reached!!")
		rd := utility.BuildErrorResponse(http.StatusForbidden, "error", "You have reached the maximum number of users for your organization plan", "Plan Limit Reached", nil)
		c.JSON(http.StatusForbidden, rd)
		return false
	}

	return true
}
