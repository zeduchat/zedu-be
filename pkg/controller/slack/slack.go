package slack

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/slack"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) SlackOauth(c *gin.Context) {
	var req models.OAuth

	err := c.ShouldBindJSON(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		if err.Error() == "user claims not found" {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to create blog", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to create blog", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	userId := userID.(string)

	accessToken, err := service.ExchangeSlackOAuthToken(base.Db.Postgresql, req, base.ExtReq, userId)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "Slack OAuth token exchange failed", err.Error(), nil, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	response := map[string]string{"access_token": accessToken}
	rd := utility.BuildSuccessResponse(http.StatusOK, "Slack OAuth token exchange successful", response)
	c.JSON(http.StatusOK, rd)
}
