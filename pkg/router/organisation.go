package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Organisation(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	organisation := organisation.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	organisationUrl := r.Group(fmt.Sprintf("%v", ApiVersion), middleware.Authorize(db.Postgresql))
	{

		organisationUrl.POST("/organisations", organisation.CreateOrganisation)
		organisationUrl.GET("/organisations/:org_id", organisation.GetOrganisation)
		organisationUrl.DELETE("/organisations/:org_id", organisation.DeleteOrganisation)
		organisationUrl.PUT("/organisations/:org_id", organisation.UpdateOrganisation)
		organisationUrl.GET("/organisations/:org_id/users", organisation.GetUsersInOrganisation)
		organisationUrl.POST("/organisations/:org_id/roles", organisation.CreateOrgRole)
		organisationUrl.GET("/organisations/:org_id/roles", organisation.GetOrgRoles)
		organisationUrl.GET("/organisations/:org_id/roles/:role_id", organisation.GetAOrgRole)
		organisationUrl.DELETE("/organisations/:org_id/roles/:role_id", organisation.DeleteOrgRole)
		organisationUrl.PUT("/organisations/:org_id/roles/:role_id", organisation.UpdateOrgRole)
		organisationUrl.PUT("/organisations/:org_id/roles/:role_id/permissions", organisation.UpdateOrgPermissions)
	}

	return r
}
