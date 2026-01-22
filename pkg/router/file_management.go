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
		fileManagementUrl.POST("/upload-folder", fileManagement.UploadFolderWithFiles)
		fileManagementUrl.GET("/file/:id", fileManagement.GetFileDetailsByID)
		fileManagementUrl.GET("/file/:id/info", fileManagement.GetFileInfo)
		fileManagementUrl.DELETE("/file/:id", fileManagement.DeleteFileDetailsByID)
		fileManagementUrl.PUT("/file/:id", fileManagement.UpdateFileName)
		fileManagementUrl.PUT("/file/:id/restore", fileManagement.RestoreFile)

		fileManagementUrl.POST("/folders", fileManagement.CreateFolder)
		fileManagementUrl.GET("/folders", fileManagement.GetFolders)
		fileManagementUrl.PUT("/folders/:id", fileManagement.UpdateFolderName)
		fileManagementUrl.DELETE("/folders/:id", fileManagement.DeleteFolder)

		fileManagementUrl.PUT("/:id/move", fileManagement.MoveFile)

		fileManagementUrl.DELETE("", fileManagement.DeleteMultipleFiles)
		fileManagementUrl.DELETE("/folders", fileManagement.DeleteMultipleFolders)

		fileManagementUrl.GET("", fileManagement.GetFiles)

		fileManagementUrl.POST("/file/:id/pin", fileManagement.PinFile)
		fileManagementUrl.DELETE("/file/:id/pin", fileManagement.UnpinFile)
		fileManagementUrl.GET("/favorites", fileManagement.GetPinnedFiles)
		fileManagementUrl.GET("/recent", fileManagement.GetRecentFiles)

		// NEW ROUTES - File Sharing
		fileManagementUrl.POST("/:id/share", fileManagement.ShareFile)
		fileManagementUrl.PUT("/shares/:id", fileManagement.UpdateFileShare)
		fileManagementUrl.DELETE("/shares/:id", fileManagement.RevokeFileShare)
		fileManagementUrl.GET("/:id/shares", fileManagement.GetFileShares)
		fileManagementUrl.POST("/access", fileManagement.AccessSharedFile)
		fileManagementUrl.PUT("/:id/access-settings", fileManagement.UpdateFileAccessSettings)
		fileManagementUrl.POST("/:id/send-dm", fileManagement.SendFileToDM)
	}

	return r
}
