package mongogrations

import (
	"errors"
	"fmt"
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

func (base *Controller) CreateCollection(c *gin.Context) {
	var req models.CreateMongoCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error",
			"Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	ids, err := mongogrations.FetchMongoAgentIDs(c)
	if err != nil {
		base.Logger.Error("Failed to fetch agent IDs", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to fetch agent IDs", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	
	collection_name := fmt.Sprintf("org_%v_agent_%v_%v", ids.OrganisationID, ids.AgentID, req.Collection)

	err = mongogrations.CreateCollection(base.Db.Mongo, collection_name)
	if err != nil {
		base.Logger.Error("Failed to create collection", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to create collection", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, fmt.Sprintf("Collection %s created successfully", req.Collection), nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) CreateEntry(c *gin.Context) {

	var req models.CreateMongoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error",
			"Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	collection_name := c.Param("collection_name")
	if collection_name == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "collection_name is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if collection_name == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "collection_name is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids, err := mongogrations.FetchMongoAgentIDs(c)
	if err != nil {
		base.Logger.Error("Failed to fetch agent IDs", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to fetch agent IDs", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	fullCollectionName := fmt.Sprintf("org_%v_agent_%v_%v", ids.OrganisationID, ids.AgentID, collection_name)

	// Call the service layer
	err = mongogrations.CreateEntry(base.Db.Mongo, fullCollectionName, req.Document)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, "Document created successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ListCollections(c *gin.Context) {
	ids, err := mongogrations.FetchMongoAgentIDs(c)
	if err != nil {
		base.Logger.Error("Failed to fetch agent IDs", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to fetch agent IDs", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	fullCollectionName := fmt.Sprintf("org_%v_agent_%v_", ids.OrganisationID, ids.AgentID)

	// Call the service layer
	results, err := mongogrations.ListCollections(base.Db.Mongo, fullCollectionName)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, "Collections retrieved successfully", results)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteCollection(c *gin.Context) {

	collection_name := c.Param("collection_name")
	if collection_name == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "collection_name is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids, err := mongogrations.FetchMongoAgentIDs(c)
	if err != nil {
		base.Logger.Error("Failed to fetch agent IDs", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to fetch agent IDs", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	fullCollectionName := fmt.Sprintf("org_%v_agent_%v_%v", ids.OrganisationID, ids.AgentID, collection_name)

	err = mongogrations.DeleteCollection(base.Db.Mongo, fullCollectionName)
	if err != nil {
		base.Logger.Error("Failed to delete collection", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to delete collection", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, fmt.Sprintf("Collection %s deleted successfully", collection_name), nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ReadEntries(c *gin.Context) {

	var req models.ReadMongoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error",
			"Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	collection_name := c.Param("collection_name")
	if collection_name == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "collection_name is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids, err := mongogrations.FetchMongoAgentIDs(c)
	if err != nil {
		base.Logger.Error("Failed to fetch agent IDs", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to fetch agent IDs", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	fullCollectionName := fmt.Sprintf("org_%v_agent_%v_%v", ids.OrganisationID, ids.AgentID, collection_name)

	// Call the service layer
	results, err := mongogrations.ReadEntries(base.Db.Mongo, fullCollectionName, req.Filter)
	if err != nil {
		base.Logger.Error("Failed to read entries", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to read entries", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Read entries successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Document(s) retrieved successfully", results)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateEntry(c *gin.Context) {

	var req models.UpdateMongoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error",
			"Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	entry_id := c.Param("entry_id")
	if entry_id == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "entry_id is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	collection_name := c.Param("collection_name")
	if collection_name == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "collection_name is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids, err := mongogrations.FetchMongoAgentIDs(c)
	if err != nil {
		base.Logger.Error("Failed to fetch agent IDs", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to fetch agent IDs", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	fullCollectionName := fmt.Sprintf("org_%v_agent_%v_%v", ids.OrganisationID, ids.AgentID, collection_name)

	// Call the service layer
	err = mongogrations.UpdateEntry(base.Db.Mongo, fullCollectionName, entry_id, req.Document)

	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, "Document Updated successfully", nil)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) DeleteEntry(c *gin.Context) {


	entry_id := c.Param("entry_id")
	if entry_id == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "entry_id is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	collection_name := c.Param("collection_name")
	if collection_name == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "collection_name is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids, err := mongogrations.FetchMongoAgentIDs(c)
	if err != nil {
		base.Logger.Error("Failed to fetch agent IDs", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to fetch agent IDs", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	fullCollectionName := fmt.Sprintf("org_%v_agent_%v_%v", ids.OrganisationID, ids.AgentID, collection_name)

	// Call the service layer
	deletedCount, err := mongogrations.DeleteEntry(base.Db.Mongo, fullCollectionName, entry_id)
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
	rd := utility.BuildSuccessResponse(http.StatusOK, "Document Deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}
