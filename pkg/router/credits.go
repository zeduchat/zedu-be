package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/credits"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Credits(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	credit := credits.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	creditUrl := r.Group(fmt.Sprintf("%v/credits", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		creditUrl.POST("/purchase", credit.PurchaseCredits)
		creditUrl.GET("/packages", credit.GetCreditPackages)
		creditUrl.GET("/usage-report/:org_id", credit.GetOrgCreditReport)
		creditUrl.GET("/transactions/:org_id", credit.GetOrgCreditTransactions)
		creditUrl.GET("/usage/:org_id", credit.GetOrgCreditUsage)

	}

	adminCreditUsage := credits.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	adminCreditUsageUrl := r.Group(fmt.Sprintf("%v/backoffice", ApiVersion), middleware.AdminAuthorize(db.Postgresql))
	{
		adminCreditUsageUrl.GET("/credits/usage", adminCreditUsage.GetAllCreditUsage)
	}

	return r
}
