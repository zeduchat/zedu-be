package mongogrations

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/mongogrations"
	"github.com/hngprojects/telex_be/utility"
	"go.mongodb.org/mongo-driver/mongo"
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
		base.Logger.Error("Failed to bind JSON", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("Validation failed")
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

	err = mongogrations.CreateCollection(base.Db.Mongo, req.CollectionName, ids)
	if err != nil {
		base.Logger.Error("Failed to create collection", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to create collection", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Collection %s created successfully", req.CollectionName)
	rd := utility.BuildSuccessResponse(http.StatusOK, fmt.Sprintf("Collection %s created successfully", req.CollectionName), nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) CreateDocument(c *gin.Context) {

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
		base.Logger.Error("collection_name is required")
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

	fullCollectionName := fmt.Sprintf("agent_%v_%v", ids.AgentID, collection_name)

	err = utility.ValidateDocument(req.Document)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error encountered while validating document fields", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Call the service layer
	err = mongogrations.CreateDocument(base.Db.Mongo, fullCollectionName, req.Document, ids)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, "Document created successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetAllDocuments(c *gin.Context) {

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

	fullCollectionName := fmt.Sprintf("agent_%v_%v", ids.AgentID, collection_name)

	// Call the service layer

	results, err := mongogrations.GetAllDocuments(base.Db.Mongo, fullCollectionName, req.Filter, ids)
	if err != nil {
		base.Logger.Error("Failed to read entries", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to read entries", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if len(results) == 0 {
		base.Logger.Error("No documents found matching the filter")
		c.JSON(http.StatusNotFound, utility.BuildErrorResponse(http.StatusNotFound, "error", "No documents found matching the filter", "No match found", results))
		return
	}

	base.Logger.Info("Read entries successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Document(s) retrieved successfully", results)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetDocument(c *gin.Context) {
	collectionName := c.Param("collection_name")
	document_id := c.Param("document_id")

	if document_id == "" {
		base.Logger.Error("document_id is required")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "document_id is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	if collectionName == "" {
		base.Logger.Error("collectionName is required")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "collectionName is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids, err := mongogrations.FetchMongoAgentIDs(c)
	if err != nil {
		base.Logger.Error("Failed to fetch agent IDs", err.Error())
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to fetch agent IDs", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	fullCollectionName := fmt.Sprintf("agent_%s_%s", ids.AgentID, collectionName)
	document, statusCode, err := mongogrations.GetDocumentByID(base.Db.Mongo, fullCollectionName, document_id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, utility.BuildErrorResponse(http.StatusNotFound, "error", "Document not found", err.Error(), nil))
			return
		}
		c.JSON(statusCode, utility.BuildErrorResponse(statusCode, "error", "Failed to retrieve document", err.Error(), nil))
		return
	}

	base.Logger.Info("Document retrieved successfully")
	c.JSON(http.StatusOK, utility.BuildSuccessResponse(http.StatusOK, "Document retrieved successfully", document))
}

func (base *Controller) UpdateDocument(c *gin.Context) {

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

	document_id := c.Param("document_id")
	if document_id == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "document_id is required", nil, nil)
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

	fullCollectionName := fmt.Sprintf("agent_%v_%v", ids.AgentID, collection_name)

	err = utility.ValidateDocument(req.Document)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error encountered while validating document fields", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Call the service layer
	err = mongogrations.UpdateDocument(base.Db.Mongo, fullCollectionName, document_id, req.Document)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Document update successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Document Updated successfully", nil)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) DeleteDocument(c *gin.Context) {

	document_id := c.Param("document_id")
	if document_id == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "document_id is required", nil, nil)
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

	fullCollectionName := fmt.Sprintf("agent_%v_%v", ids.AgentID, collection_name)

	// Call the service layer
	deletedCount, err := mongogrations.DeleteDocument(base.Db.Mongo, fullCollectionName, document_id)
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

func (base *Controller) FetchAPIKey(c *gin.Context) {
	organisation_id := c.Param("org_id")
	agent_id := c.Param("agent_id")

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		if err.Error() == "user claims not found" {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to get user claims", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to get user claims", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	userId := userID.(string)

	if organisation_id == "" {
		base.Logger.Error("organisation_id is required")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "organisation_id is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if agent_id == "" {
		base.Logger.Error("agent_id is required")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "agent_id is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.IDS{
		AgentID:        agent_id,
		OrganisationID: organisation_id,
		UserID:         userId,
	}

	response, code, err := mongogrations.FetchAPIKey(base.Db.Postgresql, ids)
	if err != nil {
		base.Logger.Error("Failed to fetch API key", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to fetch API key", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("API key fetched successfully")
	rd := utility.BuildSuccessResponse(code, "API key fetched successfully", response)
	c.JSON(code, rd)
}
