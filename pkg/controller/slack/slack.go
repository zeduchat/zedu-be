package slack

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

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

	orgId := c.Param("orgId")

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to retrieve users", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to create blog", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	userId := userID.(string)
	req.OrganisationID = orgId

	response, err := service.ExchangeSlackOAuthToken(base.Db.Postgresql, req, base.ExtReq, userId)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "Slack OAuth token exchange failed", err.Error(), nil, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Slack OAuth token exchange successful", response)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetSlackAccessToken(c *gin.Context) {
	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to retrieve user claims", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	userId := userID.(string)

	organisationID:= c.Param("orgId")

	if _, err := uuid.Parse(organisationID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to retrieve users", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	response, err := service.GetSlackAccessToken(base.Db.Postgresql, userId, organisationID)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to fetch access token info", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "slack access info fetched successfully", response)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetSlackChannels(c *gin.Context) {
	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to retrieve user claims", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	userId := userID.(string)

	organisationID := c.Query("organisation_id")
	if organisationID == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "organisation_id query param is required", "failed to fetch slack channels", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	channels, err := service.GetSlackChannels(base.Db.Postgresql, base.ExtReq, userId, organisationID)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to fetch slack channels", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Slack channels fetched successfully", channels)
	c.JSON(http.StatusOK, rd)
}
