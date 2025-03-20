package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/integrations"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Integration(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	integration := integrations.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	integrationUrl := r.Group(fmt.Sprintf("%v/integrations", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		integrationUrl.GET(":integration_id/settings", integration.GetIntegrationSettingsAllOrgs)
		integrationUrl.POST("/trigger-tick", integration.TriggerTick)
	}

	// Unauthenticated endpoint to fetch integrations
	intPage := r.Group(fmt.Sprintf("%v/integrations", ApiVersion))
	{
		intPage.GET("", integration.GetSystemIntegrationApps)
		intPage.GET(":integration_id", integration.GetSystemIntegrationApp)
	}

	external_int := r.Group(fmt.Sprintf("%v/integrations/settings", ApiVersion))
	{
		external_int.GET("", integration.GetCustomIntegrationSettingsExteranl)
		external_int.PUT("", integration.UpdateCustomIntegrationSettingsExternal)
	}

	return r
}
