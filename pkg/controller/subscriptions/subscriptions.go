package subscriptions

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/subscription"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) CreateSubscription(c *gin.Context) {

	var (
		req *models.CreateSubscriptionRequest
		env = c.Param("env")
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	subscriptionData, code, err := service.CreateSubscription(req, base.Db.Postgresql, env)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusCreated, "Subscription created successfully", subscriptionData)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) ListSubscriptions(c *gin.Context) {

	var (
		userID = c.Param("user_id")
	)

	subscriptionsData, code, err := service.ListSubscriptions(userID, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Subscriptions retrieved successfully", subscriptionsData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ModifySubscription(c *gin.Context) {

	var (
		req *models.ModifySubscriptionRequest
		env = c.Param("env")
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	subscriptionData, code, err := service.ModifySubscription(req, base.Db.Postgresql, env)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Subscription modified successfully", subscriptionData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteSubscription(c *gin.Context) {

	var (
		user_id = c.Param("user_id")
	)

	code, err := service.DeleteSubscription(user_id, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Subscription deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) CompleteSubscription(c *gin.Context) {

	var (
		session_id = c.Param("session_id")
		user_id    = c.Param("user_id")
	)

	subscriptionData, code, err, _ := service.CompleteSubscription(session_id, user_id, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Subscription completed successfully", subscriptionData)
	c.JSON(http.StatusOK, rd)
}
