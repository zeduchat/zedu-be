package mongogrations

import (
	"fmt"

	"github.com/hngprojects/telex_be/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

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
