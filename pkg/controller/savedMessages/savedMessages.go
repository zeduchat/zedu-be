package savedMessages

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/savedMessages"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) SaveThreadMessageForLater(c *gin.Context) {
	var req models.SaveThreadRequest
	orgId := c.Param("org_id")

	if _, err := uuid.Parse(orgId); err != nil {
		base.Logger.Error("Invalid organisation ID format: %v", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "unable to parse organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Error("Failed to parse request body: %v", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("Validation failed: %v", err)
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
	req.OrgId = orgId

	messageDocument, err := savedMessages.SaveThreadMessageForLater(req, base.Db.Postgresql, base.Logger)
	if err != nil {
		base.Logger.Error("Failed to save message: %v", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to save message", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Message saved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Message saved successfully", messageDocument)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) SaveReplyMessageForLater(c *gin.Context) {
	var req models.SaveMessageRequest
	orgId := c.Param("org_id")

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "unable to parse organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

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
	req.OrgId = orgId

	messageDocument, err := savedMessages.SaveReplyMessageForLater(req, base.Db.Postgresql, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to save message", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Message saved successfully", messageDocument)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetAllSavedMessages(c *gin.Context) {
	orgId := c.Param("org_id")

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "unable to parse organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	savedMessageIds := models.SavedMessageIds{
		UserID: userId,
		OrgID:  orgId,
	}

	file, err := savedMessages.GetAllSavedMessages(base.Db.Postgresql, base.Logger, savedMessageIds)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Messages not found", err.Error(), nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	base.Logger.Info("Messages retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Messages retrieved successfully", file)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteMessageByID(c *gin.Context) {
	orgId := c.Param("org_id")
	messageId := c.Param("messageId")

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "unable to parse organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(messageId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid message id format", "unable to parse message id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	savedMessageIds := models.SavedMessageIds{
		MessageID: messageId,
		UserID:    userId,
		OrgID:     orgId,
	}
	err := savedMessages.DeleteSavedMessage(base.Db.Postgresql, base.Logger, savedMessageIds)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Unable to delete message", err.Error(), nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	base.Logger.Info("Message deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Message deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}
