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
		req models.CreateSubscriptionRequest
		url = c.Request.Header.Get("Referer")
	)

	if url == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "missing URL", "missing URL", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed",
			utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	subscriptionData, code, err := service.CreateSubscription(req, base.Db.Postgresql, url)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		base.Logger.Error(err.Error())
		return
	}

	base.Logger.Info("subscription created")
	rd := utility.BuildSuccessResponse(code, "Subscription created successfully", subscriptionData)
	c.JSON(code, rd)
}

func (base *Controller) ListSubscriptions(c *gin.Context) {

	var (
		orgID = c.Param("org_id")
	)

	subscriptionsData, code, err := service.ListSubscriptions(orgID, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		base.Logger.Error(err.Error())
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Subscriptions retrieved successfully", subscriptionsData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ModifySubscription(c *gin.Context) {

	var (
		req models.ModifySubscriptionRequest
		url = c.Request.Header.Get("Referer")
	)

	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed",
			utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	subscriptionData, code, err := service.ModifySubscription(req, base.Db.Postgresql, url)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		base.Logger.Error(err.Error())
		return
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, "Subscription modified successfully", subscriptionData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteSubscription(c *gin.Context) {

	var (
		org_id = c.Param("org_id")
	)

	code, err := service.DeleteSubscription(org_id, base.Db.Postgresql)
	if err != nil {

		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		base.Logger.Error(err.Error())
		return
	}

	base.Logger.Info("subscription deleted")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Subscription deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) CompleteSubscription(c *gin.Context) {

	var (
		req models.CompleteSubscriptionRequest
	)

	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed",
			utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	subscriptionData, code, _, err := service.CompleteSubscription(req, base.Db.Postgresql, base.Db.Redis)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", "Something went wrong", err.Error(), nil)
		c.JSON(code, rd)
		base.Logger.Error(err.Error())
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Subscription completed successfully", subscriptionData)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetCurrentSubscription(c *gin.Context) {

	var (
		orgID = c.Param("org_id")
	)

	subscriptionsData, code, err := service.GetCurrentSubscription(orgID, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		base.Logger.Error(err.Error())
		return
	}

	base.Logger.Info("subscription retreived")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Current subscription retrieved successfully", subscriptionsData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetSubscriptions(c *gin.Context) {

	var (
		orgID = c.Param("org_id")
	)

	subscriptionsData, code, err := service.GetSubscriptions(orgID, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		base.Logger.Error(err.Error())
		return
	}

	base.Logger.Info("subscription retreived")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Subscriptions retrieved successfully", subscriptionsData)
	c.JSON(http.StatusOK, rd)
}
