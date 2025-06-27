package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func UpdateDocument(db *mongo.Client, collection string, id string, update map[string]any) error {

	databaseName := config.Config.MongoDB.DB_Name
	dbCollection := db.Database(databaseName).Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	ObjectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	// Convert the update map to bson.M

	bsonUpdate := bson.M(update)

	result, err := dbCollection.UpdateOne(ctx, bson.M{"_id": ObjectID}, bson.M{"$set": bsonUpdate})
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no document found with ID: %s in collection %s", id, collection)
	}

	return nil
}
