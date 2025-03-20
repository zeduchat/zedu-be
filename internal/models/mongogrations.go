package models

import (
	"github.com/hngprojects/telex_be/pkg/repository/storage/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type CreateMongoRequest struct {
	Collection string                 `json:"collection"`
	Document   map[string]interface{} `json:"document"`
}

type ReadMongoRequest struct {
	Collection string                 `json:"collection"`
	Filter     map[string]interface{} `json:"filter"`
}

type UpdateMongoRequest struct {
	Collection string                 `json:"collection"`
	ID         string                 `json:"id"`
	Document   map[string]interface{} `json:"document"`
}

type DeleteMongoRequest struct {
	Collection string `json:"collection"`
	ID         string `json:"id"`
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
