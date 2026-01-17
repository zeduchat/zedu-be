package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/avatar"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

// Avatar sets up routes for avatar management
func Avatar(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	avatarCtrl := avatar.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	avatarUrl := r.Group(fmt.Sprintf("%v/avatars", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		avatarUrl.POST("", avatarCtrl.UploadAvatar)
		avatarUrl.GET("", avatarCtrl.ListAvatars)
	}

	return r
}
