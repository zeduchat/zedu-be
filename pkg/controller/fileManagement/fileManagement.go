package fileManagement

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	services "github.com/hngprojects/telex_be/services/fileManagement"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) UploadController(c *gin.Context) {
	var req models.UploadRequest

	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	var uploadedFiles []string

	for _, fileHeader := range req.Files {
		// Open the file
		file, err := fileHeader.Open()
		if err != nil {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid file", err, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		defer file.Close()

		// Call the UploadFile service
		filePath, err := services.UploadFiles(base.Logger, file, fileHeader)
		if err != nil {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Upload failed", err, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}

		uploadedFiles = append(uploadedFiles, filePath)
	}

	base.Logger.Info("Files uploaded successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Files uploaded successfully", uploadedFiles)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) FileController(c *gin.Context) {

	objectName := c.Param("filename")

	preSignedUrl, err := services.GeneratePresignedURL(base.Logger, objectName)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to generate presigned URL", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("URL generated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "URL generated successfully", preSignedUrl)
	c.JSON(http.StatusOK, rd)
}
