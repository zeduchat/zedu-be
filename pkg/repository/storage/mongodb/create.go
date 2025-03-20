package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

var databaseName string

func GetDBName(dbName string) string {
	databaseName = dbName
	return dbName
}

func CreateEntry(db *mongo.Client, collection string, document interface{}) error {

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Second)
	defer cancel()

	dbCollection := db.Database(databaseName).Collection(collection)
	_, err := dbCollection.InsertOne(ctx, document)
	if err != nil {
		return err
	}
	return nil
}
