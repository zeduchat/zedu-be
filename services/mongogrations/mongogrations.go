package mongogrations

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func FetchMongoAgentIDs(c *gin.Context) (models.IDS, error) {
	var idmodel models.IDS

	agentID, ok := c.Get("agent_id")
	if !ok {
		return idmodel, errors.New("agent_id is required")
	}

	organisationID, ok := c.Get("org_id")
	if !ok {
		return idmodel, errors.New("organisation_id is required")
	}

	ids := models.IDS{
		AgentID:        agentID.(string),
		OrganisationID: organisationID.(string),
	}

	return ids, nil
}

func CreateCollection(db *mongo.Client, collection_name string, ids models.IDS) error {
	if collection_name == "" {
		return fmt.Errorf("collection name cannot be empty")
	}

	mongo_collection_name := fmt.Sprintf("agent_%v_%v", ids.AgentID, collection_name)

	// Call the storage layer
	err := models.CreateCollection(db, mongo_collection_name, ids)
	if err != nil {
		return err
	}

	return nil
}

func CreateDocument(db *mongo.Client, collection string, document map[string]interface{}, ids models.IDS) error {
	if len(document) == 0 {
		return fmt.Errorf("document cannot be empty")
	}

	document["agent_id"] = ids.AgentID
	document["organisation_id"] = ids.OrganisationID

	// Call the storage layer
	err := models.CreateDocument(db, collection, document)
	if err != nil {
		return err
	}

	return nil
}

func GetAllDocuments(db *mongo.Client, collection string, filter map[string]interface{}, ids models.IDS) ([]bson.M, error) {
	if collection == "" {
		return nil, fmt.Errorf("collection cannot be empty")
	}

	filter["agent_id"] = ids.AgentID
	filter["organisation_id"] = ids.OrganisationID

	// Call the storage layer
	results, err := models.GetAllDocuments(db, collection, filter)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func GetDocumentByID(db *mongo.Client, collection string, id string) (bson.M, int, error) {

	document, statusCode ,err := models.GetDocumentByID(db, collection, id)
	if err != nil {
		return nil, statusCode ,err
	}

	return document, statusCode ,nil
}

func UpdateDocument(db *mongo.Client, collection string, id string, update map[string]interface{}) error {
	if collection == "" {
		return fmt.Errorf("collection name is required")
	}

	if len(update) == 0 {
		return fmt.Errorf("update data cannot be empty")
	}

	// Call the storage layer
	err := models.UpdateDocument(db, collection, id, update)
	if err != nil {
		return err
	}

	return nil
}

func DeleteDocument(db *mongo.Client, collection string, id string) (int64, error) {
	if collection == "" {
		return 0, fmt.Errorf("collection name is required")
	}

	if id == "" {
		return 0, fmt.Errorf("id cannot be empty")
	}

	// Call the storage layer
	deletedCount, err := models.DeleteDocument(db, collection, id)
	if err != nil {
		return 0, err
	}

	return deletedCount, nil
}
