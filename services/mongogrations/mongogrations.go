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

func CreateCollection(db *mongo.Client, collection string) error {
	if collection == "" {
		return fmt.Errorf("collection name cannot be empty")
	}

	// Call the storage layer
	err := models.CreateCollection(db, collection)
	if err != nil {
		return err
	}

	return nil
}

func CreateEntry(db *mongo.Client, collection string, document map[string]interface{}) error {
	if collection == "" {
		return fmt.Errorf("collection cannot be empty")
	}

	if len(document) == 0 {
		return fmt.Errorf("document cannot be empty")
	}

	// Call the storage layer
	err := models.CreateEntry(db, collection, document)
	if err != nil {
		return err
	}

	return nil
}

func ListCollections(db *mongo.Client, prefix string) ([]string, error) {
	if prefix == "" {
		return nil, fmt.Errorf("collection name prefix cannot be empty")
	}

	// Call the storage layer
	collections, err := models.ListCollections(db, prefix)
	if err != nil {
		return nil, err
	}

	return collections, nil
}

func DeleteCollection(db *mongo.Client, collection string) error {
	// Call the storage layer
	err := models.DeleteCollection(db, collection)
	if err != nil {
		return err
	}

	return nil
}

func ReadEntries(db *mongo.Client, collection string, filter map[string]interface{}) ([]bson.M, error) {
	if collection == "" {
		return nil, fmt.Errorf("collection cannot be empty")
	}

	// Call the storage layer
	results, err := models.ReadEntries(db, collection, filter)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func GetDocumentByID(db *mongo.Client, collection string, id string) (bson.M, error) {

	document, err := models.GetDocumentByID(db, collection, id)
	if err != nil {
		return nil, err
	}

	return document, nil
}

func UpdateEntry(db *mongo.Client, collection string, id string, update map[string]interface{}) error {
	if collection == "" {
		return fmt.Errorf("collection name is required")
	}

	if len(update) == 0 {
		return fmt.Errorf("update data cannot be empty")
	}

	// Call the storage layer
	err := models.UpdateEntry(db, collection, id, update)
	if err != nil {
		return err
	}

	return nil
}

func DeleteEntry(db *mongo.Client, collection string, id string) (int64, error) {
	if collection == "" {
		return 0, fmt.Errorf("collection name is required")
	}

	if id == "" {
		return 0, fmt.Errorf("id cannot be empty")
	}

	// Call the storage layer
	deletedCount, err := models.DeleteEntry(db, collection, id)
	if err != nil {
		return 0, err
	}

	return deletedCount, nil
}
