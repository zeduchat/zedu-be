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

func DeleteEntry(db *mongo.Client, collection string, id string) (int64, error) {

	databaseName := config.Config.MongoDB.DB_Name
	dbCollection := db.Database(databaseName).Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	ObjectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return 0, err
	}
	del, err := dbCollection.DeleteOne(ctx, bson.M{"_id": ObjectID})
	deletedCount := del.DeletedCount
	if err != nil {
		return 0, err
	}
	if deletedCount == 0 {
		return 0, fmt.Errorf("no document found with ID: %s", id)
	}
	return deletedCount, err
}
