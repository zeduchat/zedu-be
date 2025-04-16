package models

import (
	"github.com/hngprojects/telex_be/pkg/repository/storage/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type CreateMongoRequest struct {
	Document map[string]interface{} `json:"document" validate:"required"`
}

type CreateMongoCollectionRequest struct {
	Collection string `json:"collection" validate:"required"`
}

type ReadMongoRequest struct {
	Filter     map[string]interface{} `json:"filter"`
}

type UpdateMongoRequest struct {
	Document   map[string]interface{} `json:"document" validate:"required"`
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

func CreateEntry(db *mongo.Client, collection string, document map[string]interface{}) error {

	// Call the storage layer
	err := mongodb.CreateEntry(db, collection, document)
	if err != nil {
		return err
	}

	return nil
}

func DeleteCollection(db *mongo.Client, collection string) error {

	// Call the storage layer
	err := mongodb.DeleteCollection(db, collection)
	if err != nil {
		return err
	}

	return nil
}

func CreateCollection(db *mongo.Client, collection string) error {

	// Call the storage layer
	err := mongodb.CreateCollection(db, collection)
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
