package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/credentials"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func CredentialRoutes(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	credCtrl := credentials.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	credGroup := r.Group(fmt.Sprintf("%v/credentials", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		credGroup.POST("", credCtrl.CreateCredential)
		credGroup.GET("/skill/:skill_id", credCtrl.GetSkillCredentials)
		credGroup.GET("/:credential_id", credCtrl.GetCredentialByID)
	}

	return r
}
