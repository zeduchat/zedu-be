package channel

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	notificationpref "github.com/hngprojects/telex_be/services/notification_pref"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) UpdateDeviceNotification(c *gin.Context) {
	var (
		req models.DeviceNotificationSettings
	)
	err := c.ShouldBindJSON(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	req.ChannelsID = c.Param("channelId")

	if _, err := uuid.Parse(req.ChannelsID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")

	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)

	req.UserID = userClaims["user_id"].(string)

	respData, code, err := notificationpref.UpdateDeviceNotification(base.Db.Postgresql, base.Logger, req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("notification settings updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "notification settings updated successfully", respData)
	c.JSON(code, rd)
}

func (base *Controller) GetChannelNotificationPref(c *gin.Context) {

	ChannelsId := c.Param("channelId")

	if _, err := uuid.Parse(ChannelsId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	claims, exists := c.Get("userClaims")

	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	UserId := userClaims["user_id"].(string)

	DeviceType := c.Param("device_type")

	if valid, msg := ValidateDeviceType(DeviceType); !valid {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid device type", errors.New(msg), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"channel_id":  ChannelsId,
		"user_id":     UserId,
		"device_type": DeviceType,
	}

	respData, err := notificationpref.GetOrCreateDeviceNotification(base.Db.Postgresql, base.Logger, ids)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("device notification fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "device notification fetched successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetUserChannelsNotificationPrefs(c *gin.Context) {

	orgId := c.Param("org_id")

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisations id format", errors.New("failed to parse organisations id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	claims, exists := c.Get("userClaims")

	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	UserId := userClaims["user_id"].(string)

	DeviceType := c.Param("device_type")

	if valid, msg := ValidateDeviceType(DeviceType); !valid {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid device type", errors.New(msg), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":      orgId,
		"user_id":     UserId,
		"device_type": DeviceType,
	}

	respData, code, err := notificationpref.GetUserChannelsNotificationPrefs(base.Db.Postgresql, base.Logger, ids)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("device notification channels pref fetched successfully")
	rd := utility.BuildSuccessResponse(code, "device notification channels pref fetched successfully", respData)
	c.JSON(code, rd)
}

func ValidateDeviceType(device string) (bool, string) {

	devices := map[string]bool{
		"web":     true,
		"desktop": true,
		"mobile":  true,
	}

	if device == "" {
		return false, "empty device type passed"
	}

	_, ok := devices[device]

	if !ok {
		return false, "invalid device passed, device supported: web, desktop, mobile"
	}

	return true, ""

}
