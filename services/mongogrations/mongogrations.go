package mongogrations

import (
	"encoding/json"
	"fmt"

	"github.com/hngprojects/telex_be/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

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

func CreateDocument(db *mongo.Client, collection string, document map[string]any, ids models.IDS) error {
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

func GetAllDocuments(db *mongo.Client, collection string, filter map[string]any, ids models.IDS) ([]bson.M, error) {
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

	document, statusCode, err := models.GetDocumentByID(db, collection, id)
	if err != nil {
		return nil, statusCode, err
	}

	return document, statusCode, nil
}

func UpdateDocument(db *mongo.Client, collection string, id string, update map[string]any) error {
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

func FetchAPIKey(db *gorm.DB, ids models.IDS) (string, int, error) {
	var cis models.CustomIntegrationsSetting

	response, code, err := cis.FetchAPIKey(db, ids)
	if err != nil {
		return "", code, err
	}

	var result struct {
		AuthCredentials struct {
			TelexAPIKey string `json:"agent_api_key"`
		} `json:"auth_credentials"`
	}

	err = json.Unmarshal([]byte(response), &result)
	if err != nil {
		return "", 500, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if result.AuthCredentials.TelexAPIKey == "" {
		return "", 404, fmt.Errorf("API key not found")
	}
	return result.AuthCredentials.TelexAPIKey, 200, nil
}
