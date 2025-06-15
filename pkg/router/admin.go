package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/admin"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Admin(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	admin := admin.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	adminAuthUrl := r.Group(fmt.Sprintf("%v/backoffice", ApiVersion), middleware.AdminAuthorize(db.Postgresql))

	{
		adminAuthUrl.GET("/admins", admin.ListAdmins)
		adminAuthUrl.POST("/admins", admin.CreateAdmin) // an admin can only be added by a superadmin when authenticated
	}

	adminUrl := r.Group(fmt.Sprintf("%v/backoffice", ApiVersion))

	{
		adminUrl.POST("/login", admin.LoginAdmin)
	}

	return r
}
