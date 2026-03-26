package user

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware/common"
	onesignalService "github.com/hngprojects/telex_be/services/onesignal"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GetUserOneSignalNotifications(c *gin.Context) {
	userClaims := common.GetAllUserClaims(c)
	userID, ok := userClaims["user_id"].(string)
	if !ok || userID == "" {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Unauthorized", nil, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	var startDate, endDate *time.Time

	if startDateStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, startDateStr)
		if err == nil {
			startDate = &parsedTime
		}
	}

	if endDateStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, endDateStr)
		if err == nil {
			endDate = &parsedTime
		}
	}

	if page < 1 {
		page = 1
	}

	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	service := onesignalService.NewOneSignalNotificationService(base.Db.Postgresql)
	respData, err := service.GetUserNotifications(userID, page, pageSize, status, startDate, endDate)
	if err != nil {
		base.Logger.Error("failed to fetch user notifications", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to fetch notifications", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Notifications retrieved successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) MarkOneSignalNotificationAsRead(c *gin.Context) {
	userClaims := common.GetAllUserClaims(c)
	userID, ok := userClaims["user_id"].(string)
	if !ok || userID == "" {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Unauthorized", nil, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	notificationID := c.Param("notification_id")
	if notificationID == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Notification ID is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	service := onesignalService.NewOneSignalNotificationService(base.Db.Postgresql)

	var notification models.OneSignalNotification
	err := base.Db.Postgresql.Where("id = ? AND user_id = ? AND deleted_at IS NULL", notificationID, userID).First(&notification).Error
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Notification not found", nil, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	err = service.MarkNotificationAsRead(notificationID)
	if err != nil {
		base.Logger.Error("failed to mark notification as read", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to mark notification as read", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Notification marked as read", gin.H{
		"notification_id": notificationID,
	})
	c.JSON(http.StatusOK, rd)
}
