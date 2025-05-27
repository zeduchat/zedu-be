package plan

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func CheckUserOrgPlanThreshold(c *gin.Context, logger *utility.Logger, db *gorm.DB, organisationID string) bool {
	var currentPlan models.Plan
	orgPlan := &models.OrganisationPlan{}

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

	currentPlan, err = orgPlan.GetPlanByOrgID(db, organisationID)

	if err != nil {
		logger.Error("unable to get organization plan data")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get organization plan data", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return false
	}

	// Check if user count exceeds plan limit (plan maximum users)
	if userCount >= int64(currentPlan.MaxUsers) {
		logger.Error("Maximum number of users for org plan reached!!")
		return false
	}

	return true
}

func CheckChannelPlanThreshold(c *gin.Context, logger *utility.Logger, db *gorm.DB, organisationID string) bool {
	var currentPlan models.Plan
	orgPlan := &models.OrganisationPlan{}

	// Count number of channels in the organization
	var channelCount int64
	err := db.Model(&models.Channels{}).
		Where("organisation_id = ?", organisationID).
		Count(&channelCount).Error

	if err != nil {
		logger.Error("Failed to count organization channels")
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Internal Server Error", "Failed to count organization channels", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return false
	}

	currentPlan, err = orgPlan.GetPlanByOrgID(db, organisationID)

	if err != nil {
		r := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Internal Server Error", err.Error(), nil)
		c.AbortWithStatusJSON(http.StatusInternalServerError, r)
		return false
	}

	// Check if channels count exceeds plan limit (plan maximum channels)
	if channelCount >= int64(currentPlan.MaxChannels) {
		logger.Error("Maximum number of channels for org plan reached!!")
		return false
	}

	return true
}
