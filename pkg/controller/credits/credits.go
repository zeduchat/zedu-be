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

func (base *Controller) TopUpOrgCredit(c *gin.Context) {

	var req models.CreditTopUpRequest

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

	// process and integrate credit top-up payment - coming soon

	organisationData, code, err := service.TopUpOrgCredit(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		base.Logger.Error(err.Error())
		return
	}

	base.Logger.Info("Credit top-up was done successfully")
	rd := utility.BuildSuccessResponse(code, "Credit top-up was done successfully", organisationData)
	c.JSON(code, rd)
}

func (base *Controller) GetOrgCreditReport(c *gin.Context) {
	org_id := c.Param("org_id")

	creditUsageReport, code, err := service.GetOrgCreditReport(org_id, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Organisation credit usage retrieved successfully", creditUsageReport)
	c.JSON(http.StatusOK, rd)

}
