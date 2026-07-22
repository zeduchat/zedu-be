package organisation

import (
	"errors"
	"net/http"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	onesignal "github.com/hngprojects/telex_be/services/onesignal"
	"github.com/hngprojects/telex_be/utility"
)


func (base *Controller) GetOneSignalNotifications(c *gin.Context) {
	orgID := c.Param("org_id")

	if _, err := uuid.Parse(orgID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userID, ok := userClaims["user_id"].(string)
	if !ok || userID == "" {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Unauthorized", nil, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	// Verify that the user belongs to the requested organization
	var membership models.OrgUserManagement
	if err := base.Db.Postgresql.Where("user_id = ? AND organisation_id = ?", userID, orgID).First(&membership).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			rd := utility.BuildErrorResponse(http.StatusForbidden, "error", "user not authorised to access this organisation", nil, nil)
			c.JSON(http.StatusForbidden, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to check organisation membership", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	// Get pagination parameters
	pagination := postgresql.GetPagination(c)
	page, limit := pagination.Page, pagination.Limit

	// Fetch active onesignal notifications
	notifs, paginationResponse, err := onesignal.GetNotificationsByUserAndOrg(base.Db.Postgresql, userID, orgID, page, limit)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to fetch onesignal notifications", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "onesignal notifications retrieved successfully", map[string]interface{}{
		"notifications": notifs,
		"pagination":    paginationResponse,
	})
	c.JSON(http.StatusOK, rd)
}
