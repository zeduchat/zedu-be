package forwardedMessage

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/forwardedMessage"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) ForwardThreadMessage(c *gin.Context) {

	var (
		req       = models.ForwardThreadMessageRequest{}
		channelID = c.Param("channelId")
	)

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Error("Failed to parse request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("Validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed",
			utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	if _, err := uuid.Parse(channelID); err != nil {
		base.Logger.Error("Invalid channel id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Error("Unable to get user claims. User not authorized")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId, idOk := userClaims["user_id"].(string)
	orgId, orgOk := userClaims["org_id"].(string)
	if !idOk || !orgOk || userId == "" {
		base.Logger.Error("User claims missing user_id or org_id")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "unauthorized", errors.New("invalid token claims"), nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}
	req.OrgId = orgId
	req.ChannelsId = channelID
	req.UserId = userId

	threadData, err := forwardedMessage.ForwardThreadMessage(base.Db, req, base.Logger, userId)
	if err != nil {
		base.Logger.Error(fmt.Sprintf("An error occurred while forwarding thread message: %v", err))
		if errors.Is(err, forwardedMessage.ErrForwardNotFound) {
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", err.Error(), nil, nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Thread message forwarded successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Thread message forwarded successfully", threadData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ForwardReplyMessage(c *gin.Context) {
	var (
		req       models.ForwardReplyMessageRequest
		channelID = c.Param("channelId")
	)

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Error("Failed to parse request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("Validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	if _, err := uuid.Parse(channelID); err != nil {
		base.Logger.Error("Invalid channel id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	req.ChannelsId = channelID

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Error("Unable to get user claims. User not authorized")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId, idOk := userClaims["user_id"].(string)
	orgId, orgOk := userClaims["org_id"].(string)
	if !idOk || !orgOk || userId == "" {
		base.Logger.Error("User claims missing user_id or org_id")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "unauthorized", errors.New("invalid token claims"), nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}
	req.UserId = userId
	req.OrgId = orgId

	response, err := forwardedMessage.ForwardReplyMessage(base.Db, req, base.Logger, userId)
	if err != nil {
		base.Logger.Error(fmt.Sprintf("An error occurred while forwarding reply message: %v", err))
		if errors.Is(err, forwardedMessage.ErrForwardNotFound) {
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", err.Error(), nil, nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Reply message forwarded successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Reply message forwarded successfully", response)
	c.JSON(http.StatusOK, rd)
}
