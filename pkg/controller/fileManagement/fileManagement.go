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

	validationErr := base.Validator.Struct(&req)
	if validationErr != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(validationErr, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	var uploadedFiles []*models.UploadedFileResponse

	for _, fileHeader := range req.Files {
		file, err := fileHeader.Open()
		if err != nil {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid file", err, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		defer file.Close()

		fileData, err := services.UploadFiles(base.Db.Postgresql, base.Logger, file, fileHeader)
		if err != nil {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Upload failed", err.Error(), nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}

		uploadedFiles = append(uploadedFiles, fileData)
	}

	base.Logger.Info("Files uploaded successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Files uploaded successfully", uploadedFiles)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetFileDetailsByID(c *gin.Context) {
	var fileModel models.UploadedFileResponse
	fileId := c.Param("id")

	file, err := fileModel.GetFileByID(base.Db.Postgresql, fileId)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "File not found", err.Error(), nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	base.Logger.Info("Files located successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "File located successfully", file)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteFileDetailsByID(c *gin.Context) {
	var fileModel models.UploadedFileResponse
	fileId := c.Param("id")

	file, err := fileModel.GetFileByID(base.Db.Postgresql, fileId)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "File not found", err.Error(), nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	minioErr := services.DeleteUploadedFiles(base.Logger, file.FileName)
	if minioErr != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Deleting failed", minioErr.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	deleteErr := fileModel.DeleteFileByID(base.Db.Postgresql, fileId)
	if deleteErr != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "File not deleted", deleteErr.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Files deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "File deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}
