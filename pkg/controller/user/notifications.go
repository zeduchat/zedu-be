package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	service "github.com/hngprojects/telex_be/services/user"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GetUserNotificationPreferences(c *gin.Context) {
	respData, code, err := service.GetUserNotificationPreferences(base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(code, "User notification preferences retrieved successfully", respData)
	c.JSON(code, rd)

}

func (base *Controller) UpdateUserNotificationSettings(c *gin.Context) {
	var (
		req = models.NotificationPreferences{}
	)

	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed",
			utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	respData, code, err := service.UpdateUserNotificationPreferences(req, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user notification preferences updated successfully")

	rd := utility.BuildSuccessResponse(http.StatusOK, "User notification preferences updated successfully", respData)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetNotificationSettings(c *gin.Context) {
	respData, code, err := service.GetEffectiveNotificationSettings(base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(code, "User notification settings retrieved successfully", respData)
	c.JSON(code, rd)
}

func (base *Controller) UpdateNotificationSetting(c *gin.Context) {
	var req models.NotificationSettingsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	respData, code, err := service.UpdateUserNotificationSetting(req, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(code, "User notification settings updated successfully", respData)
	c.JSON(code, rd)
}

func (base *Controller) ResetNotificationSettings(c *gin.Context) {
	code, err := service.ResetNotificationSettings(base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(code, "User notification settings has been reset successfully", "")
	c.JSON(code, rd)
}
