package buzz

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	buzzsvc "github.com/hngprojects/telex_be/services/buzz"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) InitiateDirectCall(c *gin.Context) {
	var req models.InitiateDirectCallRequest

	userIDInterface, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id", errors.New("invalid user id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing direct call request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for direct call request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := buzzsvc.InitiateDirectCall(base.Db, base.Logger, req.ChannelID, userID)
	if err != nil {
		base.Logger.Error("failed to initiate direct call: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("direct call initiated successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "direct call initiated successfully", resp)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) RespondToCall(c *gin.Context) {
	buzzID := c.Param("id")

	if !utility.IsValidUUID(buzzID) {
		base.Logger.Error("invalid buzz id param")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid buzz id", errors.New("invalid buzz id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userIDInterface, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id", errors.New("invalid user id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	var req models.RespondToCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing respond to call request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for respond to call request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := buzzsvc.RespondToCall(base.Db, base.Logger, buzzID, userID, req.Action)
	if err != nil {
		base.Logger.Error("failed to respond to call: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user %s responded to call %s with action: %s", userID, buzzID, req.Action)
	rd := utility.BuildSuccessResponse(http.StatusOK, "call response recorded successfully", resp)
	c.JSON(http.StatusOK, rd)
}
