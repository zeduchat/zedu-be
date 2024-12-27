package webhook

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/webhook"
	"github.com/hngprojects/telex_be/utility"
)

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

	respData, code, err := webhook.PostFeedWebhook(base.Db, base.Logger, req)

	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", "post to channel failed", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "data sent successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) PostSlugWebhook(c *gin.Context) {
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

	req.WebhookSlug = c.Param("webhook_slug")

	if err != nil {
		base.Logger.Info("invalid webhook")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "invalid webhook", err, nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	respData, code, err := webhook.PostWebhook(base.Db, base.Logger, req)

	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", "post to channel failed", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "data sent to channel successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetSlugWebhook(c *gin.Context) {
	var (
		req models.CreateWebhookHistoryRequest
	)

	req.EventName = c.Query("event_name")
	req.UserName = c.Query("username")
	req.ActionType = c.Query("action_type")
	req.Status = c.Query("status")
	req.Message = c.Query("message")
	req.WebhookSlug = c.Param("webhook_slug")
	req.AvatarURL = c.Param("avatar_url")

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	respData, code, err := webhook.PostWebhook(base.Db, base.Logger, req)

	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "data sent to channel successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetSlugWebhookQueue(c *gin.Context) {
	var (
		req          models.CreateWebhookHistoryRequest
		webhookmodel models.Webhook
		orgchanint   models.OrganisationChannelsIntegrations
	)

	req.EventName = c.Query("event_name")
	req.UserName = c.Query("username")
	req.ActionType = c.Query("action_type")
	req.Status = c.Query("status")
	req.Message = c.Query("message")
	req.WebhookSlug = c.Param("webhook_slug")
	req.AvatarURL = c.Param("avatar_url")

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	webhookResp, err := webhookmodel.CheckExistBySlug(base.Db.Postgresql, req.WebhookSlug)
	if err != nil {
		base.Logger.Error("error getting webhook")
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Webhook not found", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	req.ChannelID = webhookResp.ChannelId

	hasIntegration, _ := orgchanint.CheckHasIntegrations(base.Db.Postgresql, req.ChannelID)
	req.OrgID = orgchanint.OrgID

	if hasIntegration {
		// send to the rabbitmq service
		err = webhook.PostWebhookQueue(base.Db.Postgresql, base.Logger, req)
		if err != nil {
			base.Logger.Error("failed to post to queue")
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to post to queue", err, nil)
			c.JSON(http.StatusInternalServerError, rd)
			return
		}

		base.Logger.Info("data sent to queue successfully")
		rd := utility.BuildSuccessResponse(http.StatusOK, "data sent to queue successfully", nil)
		c.JSON(http.StatusOK, rd)
	} else {
		//send to the channel
		respData, code, err := webhook.PostWebhook(base.Db, base.Logger, req)

		if err != nil {
			rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
			c.JSON(code, rd)
			return
		}

		rd := utility.BuildSuccessResponse(http.StatusOK, "data sent to channel successfully", respData)
		c.JSON(http.StatusOK, rd)
	}
}

func (base *Controller) PostFeedWebhookQueue(c *gin.Context) {
	var (
		req models.CreateWebhookHistoryRequest
	)

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Error("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	//call the rabbitmq service
	err = webhook.PostWebhookQueue(base.Db.Postgresql, base.Logger, req)
	if err != nil {
		base.Logger.Error("failed to post to queue")
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to post to queue", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("data sent to queue successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "data sent to queue successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) PostSlugWebhookQueue(c *gin.Context) {
	var (
		req          models.CreateWebhookHistoryRequest
		webhookmodel models.Webhook
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

	req.WebhookSlug = c.Param("webhook_slug")

	if err != nil {
		base.Logger.Info("invalid webhook")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "invalid webhook", err, nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	webhookResp, err := webhookmodel.CheckExistBySlug(base.Db.Postgresql, req.WebhookSlug)
	if err != nil {
		base.Logger.Error("error getting webhook")
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Webhook not found", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	req.ChannelID = webhookResp.ChannelId

	err = webhook.PostWebhookQueue(base.Db.Postgresql, base.Logger, req)
	if err != nil {
		base.Logger.Error("failed to post to queue")
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to post to queue", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("data sent to queue successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "data sent to queue successfully", nil)
	c.JSON(http.StatusOK, rd)
}
