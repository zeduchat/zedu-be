package models

import (
	"context"
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/repository/storage/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type SchemaField struct {
    Type       string                    `bson:"type" validate:"required,oneof=string number boolean array object"`
    Required   bool                      `bson:"required"`
    AllowEmpty bool                      `bson:"allow_empty"` // For strings
    Fields     map[string]SchemaField    `bson:"fields,omitempty"` // For nested objects
}


type CreateMongoCollectionRequest struct {
	CollectionName string                    `json:"collection" validate:"required"`
	Schema     map[string]SchemaField    `json:"schema"`
}
type CreateMongoRequest struct {
	Document map[string]interface{} `json:"document" validate:"required"`
}
type ReadMongoRequest struct {
	Filter map[string]interface{} `json:"filter"`
}

type UpdateMongoRequest struct {
	Document map[string]interface{} `json:"document" validate:"required"`
}

type DeleteMongoRequest struct {
	Collection string `json:"collection" validate:"required"`
}

func ReadEntries(db *mongo.Client, collection string, filter map[string]interface{}) ([]bson.M, error) {

	// Call the storage layer
	results, err := mongodb.ReadEntries(db, collection, filter)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func GetDocumentByID(db *mongo.Client, collectionName string, document_id string) (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result bson.M
	objectID, err := primitive.ObjectIDFromHex(document_id)
	if err != nil {
		return nil, fmt.Errorf("invalid ObjectID: %v", err)
	}
	databaseName := config.Config.MongoDB.DB_Name

	err = db.Database(databaseName).Collection(collectionName).FindOne(ctx, bson.M{"_id": objectID}).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func CreateEntry(db *mongo.Client, collection string, document map[string]interface{}) error {

	// Call the storage layer
	err := mongodb.CreateEntry(db, collection, document)
	if err != nil {
		return err
	}

	return nil
}

func DeleteCollection(db *mongo.Client, ids IDS, full_collection_name string) error {
	databaseName := config.Config.MongoDB.DB_Name
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	err := db.Database(databaseName).Collection(full_collection_name).Drop(ctx)
	if err != nil {
		return err
	}

	return nil
}

func CreateCollection(db *mongo.Client, collection string) error {

	databaseName := config.Config.MongoDB.DB_Name
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := db.Database(databaseName).CreateCollection(ctx, collection)
	if err != nil {
		return err
	}
	return nil
}

func ListCollections(db *mongo.Client, collectionNamePrefix string) ([]string, error) {

	// Call the storage layer
	collections, err := mongodb.ListCollections(db, collectionNamePrefix)
	if err != nil {
		return nil, err
	}

	return collections, nil
}

func UpdateEntry(db *mongo.Client, collection string, id string, update map[string]interface{}) error {

	// Call the storage layer
	err := mongodb.UpdateEntry(db, collection, id, update)
	if err != nil {
		return err
	}

	return nil
}

func DeleteEntry(db *mongo.Client, collection string, id string) (int64, error) {

	// Call the storage layer
	deletedCount, err := mongodb.DeleteEntry(db, collection, id)
	if err != nil {
		return 0, err
	}

	return deletedCount, nil
}
