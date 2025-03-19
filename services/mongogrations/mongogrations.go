package mongogrations

import (
	"fmt"

	"github.com/hngprojects/telex_be/pkg/repository/storage/mongodb"
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
	err := mongodb.CreateEntry(db, collection, document)
	if err != nil {
		return err
	}

	return nil
}

func ReadEntries(db *mongo.Client, collection string, filter map[string]interface{}) ([]bson.M, error) {
	if collection == "" {
		return nil, fmt.Errorf("collection cannot be empty")
	}

	// Convert the filter map to bson.M

	bsonFilter := bson.M(filter)

	// Call the storage layer
	results, err := mongodb.ReadEntries(db, collection, bsonFilter)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func UpdateEntry(db *mongo.Client, collection string, filter map[string]interface{}, update map[string]interface{}) error {
	if collection == "" {
		return fmt.Errorf("collection name is required")
	}

	if len(filter) == 0 {
		return fmt.Errorf("filter cannot be empty")
	}

	if len(update) == 0 {
		return fmt.Errorf("update data cannot be empty")
	}

	// Convert the filter and update maps to bson.M
	bsonFilter := bson.M(filter)
	bsonUpdate := bson.M(update)

	// Call the storage layer
	err := mongodb.UpdateEntry(db, collection, bsonFilter, bsonUpdate)
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
	deletedCount, err := mongodb.DeleteEntry(db, collection, id)
	if err != nil {
		return 0, err
	}

	return deletedCount, nil
}
