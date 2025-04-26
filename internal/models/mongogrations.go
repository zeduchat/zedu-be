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

type CreateMongoCollectionRequest struct {
	CollectionName string `json:"collection" validate:"required"`
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

func GetAllDocuments(db *mongo.Client, collection string, filter map[string]interface{}) ([]bson.M, error) {

	// Call the storage layer
	results, err := mongodb.GetAllDocuments(db, collection, filter)
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

func CreateDocument(db *mongo.Client, collection_name string, document map[string]interface{}) error {
	var agentID string = document["agent_id"].(string)

	exists := CheckCollectionExist(db, collection_name, agentID)
	if !exists {
		return fmt.Errorf("collection with name %s does not exist for agent %s", collection_name, agentID)
	}

	// Call the storage layer
	err := mongodb.CreateDocument(db, collection_name, document)
	if err != nil {
		return err
	}

	return nil
}

func CheckCollectionExist(db *mongo.Client, collection string, agentID string) bool {
	databaseName := config.Config.MongoDB.DB_Name
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	database := db.Database(databaseName)

	agentCollections := database.Collection("agent_collections")

	// Check if the agent already has a collection
	existingCollection := agentCollections.FindOne(ctx, bson.M{"collection_name": collection, "agent_id": agentID})
	if existingCollection.Err() == nil {
		return true
	}
	if existingCollection.Err() != mongo.ErrNoDocuments {
		fmt.Printf("failed to check existing collection: %v\n", existingCollection.Err())
		return false
	}

	return false
}

func CreateCollection(db *mongo.Client, collection string, ids IDS) error {

	databaseName := config.Config.MongoDB.DB_Name
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	database := db.Database(databaseName)

	agentCollections := database.Collection("agent_collections")

	// Check if the agent already has a collection
	existingCollection := agentCollections.FindOne(ctx, bson.M{"agent_id": ids.AgentID})
	if existingCollection.Err() == nil {
		return fmt.Errorf("agent %s already has a collection", ids.AgentID)
	}
	if existingCollection.Err() != mongo.ErrNoDocuments {
		return fmt.Errorf("failed to check existing collection: %v", existingCollection.Err())
	}

	// Create the collection
	err := database.CreateCollection(ctx, collection)
	if err != nil {
		return fmt.Errorf("failed to create collection: %v", err)
	}

	_, err = database.Collection(collection).Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.M{"agent_id": 1}},
		{Keys: bson.M{"organisation_id": 1}},
	})
	if err != nil {
		return fmt.Errorf("failed to create indexes: %v", err)
	}

	// Record the collection in agent_collections
	agentCollectionDoc := bson.M{
		"agent_id":        ids.AgentID,
		"organisation_id": ids.OrganisationID,
		"collection_name": collection,
	}
	_, err = agentCollections.InsertOne(ctx, agentCollectionDoc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("agent %s already has a collection", ids.AgentID)
		}
		return fmt.Errorf("failed to record agent collection: %v", err)
	}

	return nil
}

func UpdateDocument(db *mongo.Client, collection string, id string, update map[string]interface{}) error {

	if _, ok := update["agent_id"]; ok {
		return fmt.Errorf("cannot update agent_id")
	}
	if _, ok := update["organisation_id"]; ok {
		return fmt.Errorf("cannot update organisation_id")
	}

	err := mongodb.UpdateDocument(db, collection, id, update)
	if err != nil {
		return err
	}

	return nil
}

func DeleteDocument(db *mongo.Client, collection string, id string) (int64, error) {

	// Call the storage layer
	deletedCount, err := mongodb.DeleteDocument(db, collection, id)
	if err != nil {
		return 0, err
	}

	return deletedCount, nil
}
