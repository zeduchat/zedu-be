package newsletter

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/newsletter"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) SubscribeNewsLetter(c *gin.Context) {
	var (
		req = models.NewsLetter{}
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

	err = service.NewsLetterSubscribe(&req, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Same answer for new and existing addresses, so a signup cannot reveal who is subscribed.
	base.Logger.Info("newsletter signup received")
	rd := utility.BuildSuccessResponse(http.StatusOK, "check your inbox to confirm your subscription", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ConfirmNewsLetter(c *gin.Context) {
	var req models.ConfirmNewsLetterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed",
			utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	if err := service.NewsLetterConfirm(&req, base.Db.Postgresql, base.ExtReq); err != nil {
		if errors.Is(err, service.ErrInvalidConfirmToken) {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to confirm subscription", nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		base.Logger.Error("failed to confirm newsletter subscription", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to confirm subscription", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "subscription confirmed successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UnsubscribeNewsLetter(c *gin.Context) {
	var req models.UnsubscribeNewsLetterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed",
			utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	if err := service.NewsLetterUnsubscribe(&req, base.Db.Postgresql); err != nil {
		if errors.Is(err, service.ErrInvalidUnsubscribeToken) {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to unsubscribe email", nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		base.Logger.Error("failed to unsubscribe email", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to unsubscribe email", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "email unsubscribed successfully", nil)
	c.JSON(http.StatusOK, rd)
}
