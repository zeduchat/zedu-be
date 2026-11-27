package apistatus

import (
	"io"
	"net/http"

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

const maxStatusFileBytes = 10 << 20

func (base *Controller) UpdateAPIStatus(c *gin.Context) {
	err := c.Request.ParseMultipartForm(maxStatusFileBytes)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	file, _, err := c.Request.FormFile("result_file")
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to retrieve file", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	defer file.Close()

	// Read the report in memory: nothing is written to disk, so the client's
	// file name is never used as a path and error paths leave nothing behind.
	data, err := io.ReadAll(io.LimitReader(file, maxStatusFileBytes))
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to read file", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = service.UpdateAPIStatus(base.Db.Postgresql, data)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to store api status", err, nil)
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

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  paginationResponse.TotalItems,
	}

	base.Logger.Info("api statuses retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "api statuses retrieved successfully", apiStatuses, paginationData)
	c.JSON(http.StatusOK, rd)
}
