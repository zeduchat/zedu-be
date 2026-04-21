package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/zedu"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Zedu(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	zeduCtrl := zedu.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	zeduUrl := r.Group(fmt.Sprintf("%v/zedu", ApiVersion))
	{
		zeduUrl.GET("/resources", zeduCtrl.GetResources)
	}

	return r
}
