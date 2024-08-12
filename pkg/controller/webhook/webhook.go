package webhook

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/webhook"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) GetAllWebhook(c *gin.Context) {

	usersData, paginationResponse, code, err := webhook.GetAllWebhook(base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Users retrieved successfully", usersData, paginationResponse)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetAWebhook(c *gin.Context) {

	var (
		userID = c.Param("user_id")
	)

	userData, code, err := webhook.GetAWebhook(userID, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User retrieved successfully", userData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetWebhookHistory(c *gin.Context) {

	var (
		userID = c.Param("user_id")
	)

	userData, code, err := webhook.GetWebhookHistory(userID, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User retrieved successfully", userData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) PostWebhook(c *gin.Context) {

	var (
		userID = c.Param("user_id")
	)

	userData, code, err := webhook.PostWebhook(userID, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User retrieved successfully", userData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteWebhook(c *gin.Context) {

	var (
		userID = c.Param("user_id")
	)

	userData, code, err := webhook.DeleteWebhook(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User retrieved successfully", userData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) CreateWebhook(c *gin.Context) {
	var req models.CreateWebhookRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	req.UserID = userId

	respData, code, err := webhook.CreateWebhook(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User retrieved successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ChangeWebhookStatus(c *gin.Context) {

	var (
		userID = c.Param("user_id")
	)

	userData, code, err := webhook.ChangeWebhookStatus(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User retrieved successfully", userData)
	c.JSON(http.StatusOK, rd)
}
