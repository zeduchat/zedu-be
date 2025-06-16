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

func (base *Controller) PinThreadMessage(c *gin.Context) {
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
	req.OrgId = c.Param("org_id")
	req.ChannelsId = c.Param("channel_id")

	if _, err := uuid.Parse(req.OrgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", errors.New("failed to parse organisation id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.ChannelsId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	messageDocument, err := pinnedmessages.PinThreadMessage(req, base.Db, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to pin message", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Message pinned successfully", messageDocument)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) PinReplyMessage(c *gin.Context) {
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
	req.OrgId = c.Param("org_id")
	req.ChannelsId = c.Param("channel_id")
	req.MessageID = c.Param("messageId")

	if _, err := uuid.Parse(req.OrgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", errors.New("failed to parse organisation id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.ChannelsId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.MessageID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid message id format", errors.New("failed to parse message id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	messageDocument, err := pinnedmessages.PinReplyMessage(req, base.Db, base.Logger)
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

	orgID := c.Param("org_id")
	channelID := c.Param("channel_id")

	if _, err := uuid.Parse(orgID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", errors.New("failed to parse organisation id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(channelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.PinMessageRequestIds{
		UserId:     userId,
		OrgId:      orgID,
		ChannelsId: channelID,
	}

	message, err := pinnedmessages.GetAllPinnedMessages(base.Db, base.Logger, ids)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Messages not found", err.Error(), nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	base.Logger.Info("Pinned messages retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Pinned messages retrieved successfully", message)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UnPinThreadMessage(c *gin.Context) {
	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	orgID := c.Param("org_id")
	channelID := c.Param("channel_id")
	threadID := c.Param("threadId")

	if _, err := uuid.Parse(orgID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", errors.New("failed to parse organisation id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(channelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.PinMessageRequestIds{
		UserId:     userId,
		OrgId:      orgID,
		ChannelsId: channelID,
		ThreadId:   threadID,
	}
	if err := pinnedmessages.UnPinThreadMessage(base.Db, base.Logger, ids); err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Unable to unpin thread message", err.Error(), nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	base.Logger.Info("Message unpinned successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Message unpinned successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UnPinReplyMessage(c *gin.Context) {
	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	orgID := c.Param("org_id")
	channelID := c.Param("channel_id")
	messageID := c.Param("messageId")

	if _, err := uuid.Parse(orgID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", errors.New("failed to parse organisation id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(channelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(messageID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid message id format", errors.New("failed to parse message id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.PinMessageRequestIds{
		UserId:     userId,
		OrgId:      orgID,
		ChannelsId: channelID,
		MessageID:  messageID,
	}
	if err := pinnedmessages.UnPinReplyMessage(base.Db, base.Logger, ids); err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Unable to unpin reply message", err.Error(), nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	base.Logger.Info("Message unpinned successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Message unpinned successfully", nil)
	c.JSON(http.StatusOK, rd)
}
