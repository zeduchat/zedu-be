package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

func DeleteEntry(db *mongo.Client, collection string, filter interface{}) error {

	dbCollection := db.Database("test").Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := dbCollection.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
