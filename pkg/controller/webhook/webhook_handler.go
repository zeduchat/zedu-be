package webhook

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/webhook"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) PostWebhook(c *gin.Context) {
	var (
		req models.CreateWebhookHistoryRequest
	)

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	req.WebhookSlug = c.Param("webhook_id")

	if err != nil {
		base.Logger.Info("invalid webhook")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "invalid webhook", err, nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	respData, code, err := webhook.PostWebhook(base.Db.Postgresql, base.Logger, req)

	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", "post to channel failed", err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "data sent successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetWebhook(c *gin.Context) {
	var (
		req models.CreateWebhookHistoryRequest
	)
	// http://localhost:8080/webhook?channel_id=12345&event_name=UserJoined&username=john_doe&action_type=join

	req.ChannelID = c.Query("channel_id")
	req.EventName = c.Query("event_name")
	req.UserName = c.Query("username")
	req.ActionType = c.Query("action_type")

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	if _, err := uuid.Parse(req.ChannelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", "failed to post to channel", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	respData, code, err := webhook.PostWebhook(base.Db.Postgresql, base.Logger, req)

	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", "post to channel failed", err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "data sent successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) PostFeedWebhook(c *gin.Context) {

	var (
		req models.CreateWebhookHistoryRequest
	)

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}


	if err != nil {
		base.Logger.Info("invalid webhook")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "invalid webhook", err, nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	respData, code, err := webhook.PostFeedWebhook(base.Db.Postgresql, base.Logger, req)

	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", "post to channel failed", err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "data sent successfully", respData)
	c.JSON(http.StatusOK, rd)
}
