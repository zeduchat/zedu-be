package pinnedmessages

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
	"github.com/hngprojects/telex_be/services/pinnedmessages"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) PinMessage(c *gin.Context) {
	var req models.PinMessageRequest

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

	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	req.UserId = userId
	req.ChannelsId = c.Param("channelId")

	if _, err := uuid.Parse(req.ChannelsId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	messageDocument, err := pinnedmessages.PinMessage(req, base.Db, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to pin message", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Message pinned successfully", messageDocument)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetAllPinnedMessages(c *gin.Context) {
	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	channelID := c.Param("channelId")
	if _, err := uuid.Parse(channelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	file, err := pinnedmessages.GetAllPinnedMessages(base.Db, base.Logger, channelID, userId)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Messages not found", err.Error(), nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	base.Logger.Info("Pinned messages retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Pinned messages retrieved successfully", file)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UnPinMessage(c *gin.Context) {
	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	pinnedID := c.Param("messageId")
	channelID := c.Param("channelId")

	if _, err := uuid.Parse(pinnedID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid id format", errors.New("failed to parse messageId"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(channelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := pinnedmessages.UnPinMessage(base.Db, base.Logger, pinnedID, channelID, userId)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Unable to unpin message", err.Error(), nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	base.Logger.Info("Message unpinned successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Message unpinned successfully", nil)
	c.JSON(http.StatusOK, rd)
}
