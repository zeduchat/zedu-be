package fileManagement

import (
	"github.com/gin-gonic/gin"
	"net/http"

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

// UploadController handles file uploads
// @Summary Upload multiple files
// @Description Uploads files and stores them in MinIO
// @Tags File Management
// @Accept multipart/form-data
// @Produce json
// @Param files formData file true "Files to upload"
// @Success 200 {object} map[string]interface{} "Upload success"
// @Failure 400 {object} map[string]interface{} "Invalid file format"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /files/upload-files [post]
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

// FileController retrieves a file from MinIO
// @Summary Get a file by filename
// @Description Fetches a file from storage and returns a presigned URL
// @Tags File Management
// @Produce json
// @Param filename path string true "File Name"
// @Success 200 {object} map[string]string "Presigned URL returned"
// @Failure 404 {object} map[string]interface{} "File not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /files/file/{filename} [get]
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
