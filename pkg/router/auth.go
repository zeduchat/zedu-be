package router

import (
	"fmt"
	"time"

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

	otpRequest := middleware.RateLimitByIP(db.Redis, "auth-otp-request", 10, time.Hour)
	otpVerify := middleware.RateLimitByIP(db.Redis, "auth-otp-verify", 20, 15*time.Minute)

	authUrl := r.Group(fmt.Sprintf("%v/auth", ApiVersion))
	{
		authUrl.POST("/register", auth.RegisterUser)
		authUrl.POST("/login", auth.LoginUser)
		authUrl.POST("/password-reset", otpRequest, auth.ResetPassword)
		authUrl.POST("/password-reset/verify", otpVerify, auth.VerifyResetToken)
		authUrl.POST("/magick-link", otpRequest, auth.RequestMagicLink)
		authUrl.POST("/magick-link/verify", otpVerify, auth.VerifyMagicLink)
		authUrl.POST("/email-request", otpRequest, auth.VerifyEmailReq)
		authUrl.POST("/email-request/verify", otpVerify, auth.VerifyEmailToken)
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
