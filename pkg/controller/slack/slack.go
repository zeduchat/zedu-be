package slack

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
	service "github.com/hngprojects/telex_be/services/slack"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) SlackOauth(c *gin.Context) {
	var req models.SlackTelex

	err := c.ShouldBindJSON(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	accessToken, err := service.ExchangeSlackOAuthToken(req)
    if err != nil {
        rd := utility.BuildErrorResponse(http.StatusInternalServerError, "Slack OAuth token exchange failed", err.Error(), nil, nil)
        c.JSON(http.StatusInternalServerError, rd)
        return
    }

    response := map[string]string{"access_token": accessToken}
    rd := utility.BuildSuccessResponse(http.StatusOK, "Slack OAuth token exchange successful", response, nil)
    c.JSON(http.StatusOK, rd)
}
