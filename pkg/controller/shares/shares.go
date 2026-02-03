package shares

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/shares"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Logger    *utility.Logger
	Validator *validator.Validate
	ExtReq    request.ExternalRequest
}

func (base *Controller) Create(c *gin.Context) {
	var req models.ShareRequest

	if err := c.ShouldBind(&req); err != nil {
		base.Logger.Error("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Error("Validation failed: ", err)
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Error("failed to get user: ", err)
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", err.Error(), "failed to get user", nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	share, err := service.CreateShares(base.Db.Postgresql, req, userID.(string))
	if err != nil {
		base.Logger.Error("failed to create share: ", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to create share", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("share created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "share created successfully", share)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) GetMyShares(c *gin.Context) {
	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Error("failed to get user: ", err)
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", err.Error(), "failed to get user", nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	shares, paginationResponse, err := service.GetUserShares(base.Db.Postgresql, c, userID.(string))
	if err != nil {
		base.Logger.Error("failed to fetch shares: ", err)
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "failed to fetch shares", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  paginationResponse.TotalItems,
	}

	base.Logger.Info("shares retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "shares retrieved successfully", shares, paginationData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetShare(c *gin.Context) {
	id := c.Param("id")

	share, err := service.GetShareByID(base.Db.Postgresql, id)
	if err != nil {
		base.Logger.Error("share not found: ", err)
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "share not found", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	base.Logger.Info("share retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "share retrieved successfully", share)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) Delete(c *gin.Context) {
	id := c.Param("id")

	err := service.DeleteShares(base.Db.Postgresql, id)
	if err != nil {
		base.Logger.Error("failed to delete shares: ", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to delete share", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("share deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "share deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetPerformance(c *gin.Context) {
	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Error("failed to get the user: ", err)
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", err.Error(), "failed to get user", nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	performance, err := service.GetSharePerformance(base.Db.Postgresql, c, userID.(string))
	if err != nil {
		base.Logger.Error("failed to get performance: ", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to get performance", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("share performance retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "share performance retrieved successfully", performance)
	c.JSON(http.StatusOK, rd)
}
