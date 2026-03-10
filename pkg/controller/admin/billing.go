package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GetSubscriptionBillingStats(c *gin.Context) {
	stats, err := models.GetSubscriptionBillingStats(base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to fetch subscription billing stats", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to fetch subscription billing stats", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Subscription billing stats retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Subscription billing stats retrieved successfully", stats)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetPlansFiltered(c *gin.Context) {
	status := c.Query("status")
	search := c.Query("search")

	plans, err := models.GetPlansFiltered(base.Db.Postgresql, status, search)
	if err != nil {
		base.Logger.Error("Failed to fetch plans", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to fetch plans", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Plans retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Plans retrieved successfully", plans)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) CreatePlan(c *gin.Context) {
	var req models.CreatePlanRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Error("Validation failed", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	plan, err := models.CreatePlan(base.Db.Postgresql, req)
	if err != nil {
		base.Logger.Error("Failed to create plan", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to create plan", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Plan created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Plan created successfully", plan)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) UpdatePlan(c *gin.Context) {
	planID := c.Param("plan_id")

	if _, err := uuid.Parse(planID); err != nil {
		base.Logger.Error("Invalid plan ID format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid plan ID format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	var req models.UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	plan, err := models.UpdatePlan(base.Db.Postgresql, planID, req)
	if err != nil {
		base.Logger.Error("Failed to update plan", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to update plan", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Plan updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Plan updated successfully", plan)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeletePlan(c *gin.Context) {
	planID := c.Param("plan_id")

	if _, err := uuid.Parse(planID); err != nil {
		base.Logger.Error("Invalid plan ID format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid plan ID format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := models.DeletePlan(base.Db.Postgresql, planID); err != nil {
		base.Logger.Error("Failed to delete plan", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Plan deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Plan deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) CreateAICreditPackage(c *gin.Context) {
	var req models.CreateAICreditPackageRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Error("Validation failed", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	pkg, err := models.CreateAICreditPackage(base.Db.Postgresql, req)
	if err != nil {
		base.Logger.Error("Failed to create AI credit package", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to create AI credit package", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("AI credit package created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "AI credit package created successfully", pkg)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) UpdateAICreditPackage(c *gin.Context) {
	packageID := c.Param("package_id")

	if _, err := uuid.Parse(packageID); err != nil {
		base.Logger.Error("Invalid package ID format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid package ID format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	var req models.UpdateAICreditPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	pkg, err := models.UpdateAICreditPackage(base.Db.Postgresql, packageID, req)
	if err != nil {
		base.Logger.Error("Failed to update AI credit package", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to update AI credit package", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("AI credit package updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "AI credit package updated successfully", pkg)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteAICreditPackage(c *gin.Context) {
	packageID := c.Param("package_id")

	if _, err := uuid.Parse(packageID); err != nil {
		base.Logger.Error("Invalid package ID format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid package ID format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := models.DeleteAICreditPackage(base.Db.Postgresql, packageID); err != nil {
		base.Logger.Error("Failed to delete AI credit package", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("AI credit package deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "AI credit package deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetAllCreditTransactions(c *gin.Context) {
	transactions, paginationResponse, err := models.GetAllCreditTransactions(base.Db.Postgresql, c)
	if err != nil {
		base.Logger.Error("Failed to fetch credit transactions", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to fetch credit transactions", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  paginationResponse.TotalItems,
	}

	base.Logger.Info("Credit transactions retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Credit transactions retrieved successfully", transactions, paginationData)
	c.JSON(http.StatusOK, rd)
}
