package buzz

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/services/buzz"
	"github.com/hngprojects/telex_be/utility"
)

// SendReaction handles sending ephemeral reactions in a buzz call
func (base *Controller) SendReaction(c *gin.Context) {
	var req models.SendBuzzReactionRequest

	userIDInterface, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		base.Logger.Error("user_id is not of type string")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "user_id is not of type string", errors.New("user_id is not of type string"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Get buzz ID from URL path
	buzzID := c.Param("id")
	if buzzID == "" {
		base.Logger.Info("buzz_id is required")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "buzz_id is required", errors.New("buzz_id is required"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	req.BuzzID = buzzID

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing reaction request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for reaction request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	statusCode, err := buzz.SendBuzzReaction(base.Db, base.Logger, req, userID)
	if err != nil {
		base.Logger.Error("failed to send buzz reaction: %v", err)
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), err, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("user %s sent reaction in buzz %s", userID, buzzID)
	rd := utility.BuildSuccessResponse(statusCode, "reaction sent successfully", nil)
	c.JSON(statusCode, rd)
}

// UpdateSticker handles updating a participant's status sticker in a buzz call
func (base *Controller) UpdateSticker(c *gin.Context) {
	var req models.BuzzStickerUpdateRequest

	userIDInterface, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		base.Logger.Error("user_id is not of type string")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "user_id is not of type string", errors.New("user_id is not of type string"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Get buzz ID from URL path
	buzzID := c.Param("id")
	if buzzID == "" {
		base.Logger.Info("buzz_id is required")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "buzz_id is required", errors.New("buzz_id is required"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	req.BuzzID = buzzID

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing sticker request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for sticker request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, statusCode, err := buzz.UpdateBuzzSticker(base.Db, base.Logger, req, userID)
	if err != nil {
		base.Logger.Error("failed to update buzz sticker: %v", err)
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), err, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("user %s updated sticker in buzz %s", userID, buzzID)
	rd := utility.BuildSuccessResponse(statusCode, "sticker updated successfully", resp)
	c.JSON(statusCode, rd)
}
