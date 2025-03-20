package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func ReadEntries(db *mongo.Client, collection string, filter map[string]interface{}) ([]bson.M, error) {

	dbCollection := db.Database(databaseName).Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Convert the filter map to bson.M

	bsonFilter := bson.M(filter)
	var results []bson.M
	cursor, err := dbCollection.Find(ctx, bsonFilter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
