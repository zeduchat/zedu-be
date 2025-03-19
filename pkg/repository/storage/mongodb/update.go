package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func UpdateEntry(db *mongo.Client, collection string, filter interface{}, document interface{}) error {

	dbCollection := db.Database(databaseName).Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_, err := dbCollection.UpdateMany(ctx, filter, bson.M{"$set": document})
	if err != nil {
		return err
	}

	return nil
}
