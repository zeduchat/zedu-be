package mongogrations

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/mongogrations"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) CreateEntry(c *gin.Context) {

	var req models.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Call the service layer
	err := mongogrations.CreateEntry(base.Db.Mongo, req.Collection, req.Document)

	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, "Document created successfully", nil, nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ReadEntries(c *gin.Context) {

	var req models.ReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Call the service layer
	results, err := mongogrations.ReadEntries(base.Db.Mongo, req.Collection, req.Filter)

	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, "Document retrieved successfully", results, nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateEntry(c *gin.Context) {

	var req models.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Call the service layer
	err := mongogrations.UpdateEntry(base.Db.Mongo, req.Collection, req.Filter, req.Document)

	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, "Document Updated successfully", nil, nil)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) DeleteEntry(c *gin.Context) {

	var req models.DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Call the service layer
	deletedCount, err := mongogrations.DeleteEntry(base.Db.Mongo, req.Collection, req.Id)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	if deletedCount == 0 {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "failed to delete", errors.New("failed to delete"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, "Document Deleted successfully", nil, nil)
	c.JSON(http.StatusOK, rd)
}
