package mongodb

import (
	"context"
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateEntry(db *mongo.Client, collection string, document interface{}) error {
	databaseName := config.Config.MongoDB.DB_Name
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Second)
	defer cancel()

	dbCollection := db.Database(databaseName).Collection(collection)
	_, err := dbCollection.InsertOne(ctx, document)
	if err != nil {
		return err
	}
	return nil
}
