package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Auth(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	auth := auth.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	authUrl := r.Group(fmt.Sprintf("%v/auth", ApiVersion))
	{
		authUrl.POST("/register", auth.RegisterUser)
		authUrl.POST("/login", auth.LoginUser)
		authUrl.POST("/password-reset", auth.ResetPassword)
		authUrl.POST("/password-reset/verify", auth.VerifyResetToken)
		authUrl.POST("/magick-link", auth.RequestMagicLink)
		authUrl.POST("/magick-link/verify", auth.VerifyMagicLink)
		authUrl.POST("/email-request", auth.VerifyEmailReq)
		authUrl.POST("/email-request/verify", auth.VerifyEmailToken)
		authUrl.POST("/google", auth.GoogleLogin)
		authUrl.POST("/apple", auth.AppleLogin)
	}

	authUrlSec := r.Group(
		fmt.Sprintf("%v/auth", ApiVersion),
		middleware.Authorize(db.Postgresql),
	)

	{
		authUrlSec.POST("/logout", auth.LogoutUser)
		authUrlSec.GET("/onboard-status", auth.GetOnboardStatus)
		authUrlSec.PUT("/change-password", auth.ChangePassword)
		authUrlSec.PUT("/onboard-status", auth.UpdateOnboardStatus)
	}

	return r
}
