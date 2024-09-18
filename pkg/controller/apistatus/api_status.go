package apistatus

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/apiStatus"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) UpdateAPIStatus(c *gin.Context) {
	err := c.Request.ParseMultipartForm(10 << 20)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	file, header, err := c.Request.FormFile("result_file")
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to retrieve file", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	defer file.Close()

	tempDir := "./tmp"
	err = os.MkdirAll(tempDir, os.ModePerm)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to create temp directory", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	tempFilePath := filepath.Join(tempDir, header.Filename)

	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to save file", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	defer tempFile.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to read file", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	_, err = tempFile.Write(fileContent)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to write to temp file", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	err = tempFile.Close()
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to close temp file", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	data, err := os.ReadFile(tempFilePath)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to read file", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	err = service.UpdateAPIStatus(base.Db.Postgresql, data)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to store api status", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	err = os.Remove(tempFilePath)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to delete file", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("api status stored successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "api status stored successfully", nil)

	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetAPIStatus(c *gin.Context) {
	apiStatuses, paginationResponse, err := service.GetAPIStatus(base.Db.Postgresql, c)

	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "failed to fetch api statuses", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	paginationData := map[string]interface{}{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(apiStatuses),
	}

	base.Logger.Info("api statuses retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "api statuses retrieved successfully", apiStatuses, paginationData)
	c.JSON(http.StatusOK, rd)
}
