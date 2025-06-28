package credits

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/credits"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) PurchaseCredits(c *gin.Context) {

	var req models.CreditTopUpRequest
	var url = c.Request.Header.Get("Referer")

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

	// process and integrate credit top-up payment
	checkoutSession, code, err := service.PurchaseCredits(req, base.Db.Postgresql, url)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		base.Logger.Error(err.Error())
		return
	}

	base.Logger.Info("Checkout Session URL generated succesfully")
	rd := utility.BuildSuccessResponse(code, "Checkout Session URL generated succesfully", checkoutSession)
	c.JSON(code, rd)
}

func (base *Controller) GetOrgCreditReport(c *gin.Context) {
	org_id := c.Param("org_id")

	creditUsageReport, code, err := models.GetOrgCreditReport(org_id, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Organisation credit usage retrieved successfully", creditUsageReport)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetOrgCreditTransactions(c *gin.Context) {
	org_id := c.Param("org_id")

	credits_transactions, paginationResponse, err := models.GetOrgCreditTransactions(org_id, base.Db.Postgresql, c)
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
		"total_items":  len(credits_transactions),
	}

	base.Logger.Info("credit transactions retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "credit transactions retrieved successfully.", credits_transactions, paginationData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetOrgCreditUsage(c *gin.Context) {
	org_id := c.Param("org_id")

	credits_usage, paginationResponse, err := models.GetOrgCreditUsage(org_id, base.Db.Postgresql, c)
	if err != nil {
		base.Logger.Error("Failed to fetch credit usage", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to fetch credit usage", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(credits_usage),
	}

	base.Logger.Info("credit usage retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "credit usage retrieved successfully.", credits_usage, paginationData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetCreditPackages(c *gin.Context) {
	creditPackages, code, err := models.GetCreditPackages(base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Credit packages retrieved successfully", creditPackages)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetAllCreditUsage(c *gin.Context) {
	credits_usage, paginationResponse, err := models.GetAllCreditUsage(base.Db.Postgresql, c)
	if err != nil {
		base.Logger.Error("Failed to fetch credit usage", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to fetch credit usage", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(credits_usage),
	}

	base.Logger.Info("credit usage retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "credit usage retrieved successfully.", credits_usage, paginationData)
	c.JSON(http.StatusOK, rd)
}
