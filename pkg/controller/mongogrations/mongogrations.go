package mongogrations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
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
	type CreateRequest struct {
		Collection string                 `json:"collection"`
		Document   map[string]interface{} `json:"document"`
	}

	var req CreateRequest
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
	type ReadRequest struct {
		Collection string                 `json:"collection"`
		Filter     map[string]interface{} `json:"filter"`
	}

	var req ReadRequest
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
	type UpdateRequest struct {
		Collection string                 `json:"collection"`
		Filter     map[string]interface{} `json:"filter"`
		Document   map[string]interface{} `json:"document"`
	}

	var req UpdateRequest
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
	type DeleteRequest struct {
		Collection string `json:"collection"`
		Id         string `json:"id"`
	}

	var req DeleteRequest
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
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, "Document Deleted successfully", nil, nil)
	c.JSON(http.StatusOK, rd)
}
