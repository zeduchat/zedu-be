package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func UpdateEntry(db *mongo.Client, collection string, id string, update map[string]interface{}) error {

	dbCollection := db.Database(databaseName).Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	ObjectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	// Convert the update map to bson.M

	bsonUpdate := bson.M(update)

	_, err = dbCollection.UpdateOne(ctx, bson.M{"_id": ObjectID}, bson.M{"$set": bsonUpdate})
	if err != nil {
		return err
	}

	return nil
}
