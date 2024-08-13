package webhook

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/webhook"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) GetAllWebhook(c *gin.Context) {

	channelId := c.Param("channel_id")

	if _, err := uuid.Parse(channelId); err != nil {
		base.Logger.Info("error parsing channel id")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid org id format", errors.New("failed to parse org id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	respData, additionalInfo, code, err := webhook.GetAllWebhook(base.Db.Postgresql, c, channelId)
	if err != nil {
		base.Logger.Info("error fetching webhooks")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	response := gin.H{
		"webhooks":        respData,
		"additional_info": additionalInfo,
	}

	base.Logger.Info("webhooks fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "webhooks fetched successfully", response)
	c.JSON(code, rd)
}

func (base *Controller) GetWebhookHistory(c *gin.Context) {

	var (
		channelId string
		req       models.GetWebhookHistoryRequest
	)

	channelId = c.Param("channel_id")
	WebhookID := c.Param("webhook_id")

	if _, err := uuid.Parse(channelId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", "failed to delete webhook", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	req.UserID = userId
	req.ChannelID = channelId
	req.WebhookID = WebhookID

	respData, additionalInfo, code, err := webhook.GetWebhookHistory(req, c, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	response := gin.H{
		"webhooks_history": respData,
		"additional_info":  additionalInfo,
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "webhook history retrived successfully", response)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteWebhook(c *gin.Context) {

	var (
		req       models.DeleteWebhookRequest
		channelId string
		webhookId string
	)

	channelId = c.Param("channel_id")
	webhookId = c.Param("webhook_id")

	if _, err := uuid.Parse(channelId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", "failed to delete webhook", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	req.UserID = userId
	req.ChannelID = channelId
	req.WebhookID = webhookId

	code, err := webhook.DeleteWebhook(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}
	base.Logger.Info("webhook deleted successfully")
	rd := utility.BuildSuccessResponse(code, "webhook deleted successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) UpdateWebhook(c *gin.Context) {

	var (
		req       models.UpdateWebhookRequest
		channelId string
		webhookID string
	)

	channelId = c.Param("channel_id")
	webhookID = c.Param("webhook_id")

	if _, err := uuid.Parse(channelId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", "failed to updating webhook", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

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

	req.UserID = userId
	req.ChannelID = channelId
	req.WebhookID = webhookID

	resp, code, err := webhook.UpdateWebhook(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}
	base.Logger.Info("webhook updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "webhook updated successfully", resp)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) CreateWebhook(c *gin.Context) {
	var (
		req       models.CreateWebhookRequest
		channelId string
	)

	channelId = c.Param("channel_id")

	if _, err := uuid.Parse(channelId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", "failed to create webhook", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

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

	req.UserID = userId
	req.ChannelID = channelId

	respData, code, err := webhook.CreateWebhook(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}
	base.Logger.Info("webhook created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "webhook created successfully", respData)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) ChangeWebhookStatus(c *gin.Context) {

	var (
		req models.ChangeWebhookStatusRequest
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

	channelId := c.Param("channel_id")

	if _, err := uuid.Parse(channelId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", "failed to create webhook", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	req.UserID = userId
	req.ChannelID = channelId

	userData, code, err := webhook.ChangeWebhookStatus(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("webhook status updated successfully")

	rd := utility.BuildSuccessResponse(http.StatusOK, "webhook status updated successfully", userData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) PostWebhook(c *gin.Context) {
	var (
		req models.ChangeWebhookStatusRequest
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

	channelId := c.Param("channel_id")

	if _, err := uuid.Parse(channelId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", "failed to create webhook", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	req.UserID = userId
	req.ChannelID = channelId

	rd := utility.BuildSuccessResponse(http.StatusOK, "update successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetWebhook(c *gin.Context) {
	var (
		req models.ChangeWebhookStatusRequest
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

	channelId := c.Param("channel_id")

	if _, err := uuid.Parse(channelId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", "failed to create webhook", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	req.UserID = userId
	req.ChannelID = channelId

	rd := utility.BuildSuccessResponse(http.StatusOK, "update successfully", nil)
	c.JSON(http.StatusOK, rd)
}
