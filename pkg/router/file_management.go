package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/fileManagement"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func FileManagement(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	fileManagement := fileManagement.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	fileManagementUrl := r.Group(fmt.Sprintf("%v/files", ApiVersion), middleware.Authorize(db.Postgresql))

	{
		fileManagementUrl.POST("/upload-files", fileManagement.UploadController)
		fileManagementUrl.GET("/file/:id", fileManagement.GetFileDetailsByID)
		fileManagementUrl.DELETE("/file/:id", fileManagement.DeleteFileDetailsByID)
		fileManagementUrl.PUT("/file/:id", fileManagement.UpdateFileName)

		fileManagementUrl.POST("/folders", fileManagement.CreateFolder)
		fileManagementUrl.GET("/folders", fileManagement.GetFolders)
		fileManagementUrl.DELETE("/folders/:id", fileManagement.DeleteFolder)

		fileManagementUrl.PUT("/:id/move", fileManagement.MoveFile)

		fileManagementUrl.GET("", fileManagement.GetFiles)
	}

	return r
}
