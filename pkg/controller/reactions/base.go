package reactions

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
	"github.com/hngprojects/telex_be/services/reactions"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) CreateReaction(c *gin.Context) {
	var req models.ReactionRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Error("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("Validation failed", err)
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

	req.UserID = userId
	req.ChannelsID = c.Param("channel_id")

	if _, err := uuid.Parse(req.ChannelsID); err != nil {
		base.Logger.Error("Invalid Channel Id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if req.Type == "reply" && req.MessageID == "" {
		base.Logger.Error("Invalid message Id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid or missing message id format", errors.New("failed to parse message id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	code, err := reactions.CreateReaction(req, base.Db, base.Logger)
	if err != nil {
		base.Logger.Error("Failed to add reaction", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to add reaction", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(code, "Reaction updated successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) GetThreadReactions(c *gin.Context) {
	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)
	orgId := userClaims["org_id"].(string)

	threadID := c.Param("thread_id")

	if _, err := uuid.Parse(threadID); err != nil {
		base.Logger.Error("Invalid thread Id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid thread id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	reactionID := c.Param("reaction_id")

	if _, err := uuid.Parse(reactionID); err != nil {
		base.Logger.Error("Invalid reaction Id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid reaction id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.IDS{
		UserID:         userId,
		ReactionID:     reactionID,
		ThreadID:       threadID,
		Type:           "thread",
		OrganisationID: orgId,
	}

	code, resp, err := reactions.GetReactionUsernames(base.Db, base.Logger, ids)
	if err != nil {
		base.Logger.Error("an error occured", err)
		rd := utility.BuildErrorResponse(code, "error", "an error occured", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Reaction retrieved successfully")
	rd := utility.BuildSuccessResponse(code, "Reactions retrieved successfully", gin.H{"usernames": resp})
	c.JSON(code, rd)
}

func (base *Controller) GetReplyReactions(c *gin.Context) {
	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)
	orgId := userClaims["org_id"].(string)

	messageID := c.Param("message_id")

	if _, err := uuid.Parse(messageID); err != nil {
		base.Logger.Error("Invalid message Id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid message id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	reactionID := c.Param("reaction_id")

	if _, err := uuid.Parse(reactionID); err != nil {
		base.Logger.Error("Invalid reaction Id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid reaction id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.IDS{
		UserID:         userId,
		ReactionID:     reactionID,
		MessageID:      messageID,
		Type:           "reply",
		OrganisationID: orgId,
	}

	code, resp, err := reactions.GetReactionUsernames(base.Db, base.Logger, ids)
	if err != nil {
		base.Logger.Error("an error occured", err)
		rd := utility.BuildErrorResponse(code, "error", "an error occured", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Reaction retrieved successfully")
	rd := utility.BuildSuccessResponse(code, "Reactions retrieved successfully", gin.H{"usernames": resp})
	c.JSON(code, rd)
}

func (base *Controller) DeleteThreadReactions(c *gin.Context) {
	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	threadID := c.Param("thread_id")

	if _, err := uuid.Parse(threadID); err != nil {
		base.Logger.Error("Invalid thread Id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid thread id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	reactionID := c.Param("reaction_id")

	if _, err := uuid.Parse(reactionID); err != nil {
		base.Logger.Error("Invalid reaction Id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid reaction id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.IDS{
		UserID:     userId,
		ReactionID: reactionID,
		ThreadID:   threadID,
		Type:       "thread",
	}

	code, err := reactions.DeleteReaction(base.Db, base.Logger, ids)
	if err != nil {
		base.Logger.Error("an error occured", err)
		rd := utility.BuildErrorResponse(code, "error", "an error occured", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Reaction deleted successfully")
	rd := utility.BuildSuccessResponse(code, "Reactions deleted successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) DeleteReplyReactions(c *gin.Context) {
	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	messageID := c.Param("message_id")

	if _, err := uuid.Parse(messageID); err != nil {
		base.Logger.Error("Invalid message Id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid message id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	reactionID := c.Param("reaction_id")

	if _, err := uuid.Parse(reactionID); err != nil {
		base.Logger.Error("Invalid reaction Id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid reaction id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.IDS{
		UserID:     userId,
		ReactionID: reactionID,
		MessageID:  messageID,
		Type:       "reply",
	}

	code, err := reactions.DeleteReaction(base.Db, base.Logger, ids)
	if err != nil {
		base.Logger.Error("an error occured", err)
		rd := utility.BuildErrorResponse(code, "error", "an error occured", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Reaction deleted successfully")
	rd := utility.BuildSuccessResponse(code, "Reactions deleted successfully", nil)
	c.JSON(code, rd)
}
