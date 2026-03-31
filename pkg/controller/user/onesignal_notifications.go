package user

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/middleware/common"
	onesignalService "github.com/hngprojects/telex_be/services/onesignal"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GetUserOneSignalNotifications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	c.Set("page", page)
	c.Set("page_size", pageSize)

	respData, code, err := onesignalService.GetUserNotifications(base.Db.Postgresql, c)
	if err != nil {
		base.Logger.Error("failed to fetch user notifications", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(code, "Notifications retrieved successfully", respData)
	c.JSON(code, rd)
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

	notificationIDResponse, code, err := onesignalService.MarkNotificationAsRead(base.Db.Postgresql, notificationID, userID)
	if err != nil {
		base.Logger.Error("failed to mark notification as read", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(code, "Notification marked as read", gin.H{
		"notification_id": notificationIDResponse,
	})
	c.JSON(code, rd)
}
