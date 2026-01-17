package test_avatar

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/avatar"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/minio"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func SetupAvatarTestRouter() (*gin.Engine, *avatar.Controller, *auth.Controller, *storage.Database) {
	gin.SetMode(gin.TestMode)

	logger := tests.Setup()

	db := storage.Connection()

	minio.ConnectToMinio(logger, config.Config.Minio)

	validatorRef := validator.New()

	authController := &auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	avatarController := &avatar.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	SetupAvatarRoutes(r, avatarController, db)
	return r, avatarController, authController, db
}

func GetTestLogger() *utility.Logger {
	return tests.Setup()
}

func SetupAvatarRoutes(r *gin.Engine, avatarController *avatar.Controller, db *storage.Database) {
	avatarUrl := r.Group("/api/v1/avatars", middleware.Authorize(db.Postgresql))
	{
		avatarUrl.POST("", avatarController.UploadAvatar)
		avatarUrl.GET("", avatarController.ListAvatars)
	}
}
