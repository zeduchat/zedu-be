package user

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware/common"
	fcmtokens "github.com/hngprojects/telex_be/services/fcmTokens"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) UpdateVoIPToken(c *gin.Context) {
	var req models.VoIPTokenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", nil, nil) 
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	userClaims := common.GetAllUserClaims(c)
	userID, ok := userClaims["user_id"].(string)

	if !ok || userID == "" {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Unauthorized", nil, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	if req.VoIPToken == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "VoIP token is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	user := &models.User{}
	fetchedUser, err := user.GetUserByID(base.Db.Postgresql, userID)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "User not found", nil, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	if fetchedUser.ID == "" {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "User not found", nil, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	statusCode, err := fcmtokens.CreateVoIPToken(req, userID, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("failed to update VoIP push token", err)
		rd := utility.BuildErrorResponse(statusCode, "error", "Failed to update VoIP token", nil, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("VoIP token updated successfully for user " + userID)
	rd := utility.BuildSuccessResponse(http.StatusOK, "VoIP token updated successfully", gin.H{
		"user_id":    userID,
		"voip_token": req.VoIPToken,
	})
	c.JSON(http.StatusOK, rd)
}
