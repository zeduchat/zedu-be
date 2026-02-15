package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/profile"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Profile(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	userProfile := profile.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	profileUrl := r.Group(fmt.Sprintf("%v/profile", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		profileUrl.POST("/change-status", userProfile.ChangeProfileStatus)
		profileUrl.GET("", userProfile.GetUserProfile)
		profileUrl.GET("/:user_id", userProfile.GetUserProfile)
		profileUrl.PATCH("", middleware.CheckIsDeactivated(db.Postgresql), userProfile.UpdateProfile)
		profileUrl.DELETE("/image", userProfile.DeleteUserProfileImage)
		profileUrl.POST("/presence", userProfile.ChangeProfilePresence)
		profileUrl.GET("/presence/:user_id", userProfile.GetProfilePresence)
	}

	return r
}
