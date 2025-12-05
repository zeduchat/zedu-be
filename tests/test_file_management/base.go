package test_file_management

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/fileManagement"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/minio"
	"github.com/hngprojects/telex_be/tests"
)

func SetupFileManagementTestRouter() (*gin.Engine, *fileManagement.Controller, *auth.Controller, *storage.Database) {
	gin.SetMode(gin.TestMode)

	logger := tests.Setup()

	db := storage.Connection()

	// override SSL for testing to use HTTP instead of HTTPS
	config.Config.Minio.UseSSL = false

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

	fileController := &fileManagement.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	SetupFileManagementRoutes(r, fileController, db)
	return r, fileController, authController, db
}

func SetupFileManagementRoutes(r *gin.Engine, fileController *fileManagement.Controller, db *storage.Database) {
	fileUrl := r.Group("/api/v1/files", middleware.Authorize(db.Postgresql))

	{
		fileUrl.POST("/folders", fileController.CreateFolder)
		fileUrl.GET("/folders", fileController.GetFolders)
		fileUrl.PUT("/folders/:id", fileController.UpdateFolderName)
		fileUrl.DELETE("/folders/:id", fileController.DeleteFolder)
		fileUrl.PUT("/:id/move", fileController.MoveFile)
		fileUrl.POST("/upload-files", fileController.UploadController)
		fileUrl.GET("/file/:id/info", fileController.GetFileInfo)
		fileUrl.POST("/upload-folder", fileController.UploadFolderWithFiles)
		fileUrl.DELETE("/file/:id", fileController.DeleteFileDetailsByID)
		fileUrl.PUT("/file/:id/restore", fileController.RestoreFile)
		fileUrl.DELETE("", fileController.DeleteMultipleFiles)
		fileUrl.DELETE("/folders", fileController.DeleteMultipleFolders)

		fileUrl.GET("", fileController.GetFiles)
	}
}
