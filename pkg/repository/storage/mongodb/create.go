package mongodb

import (
	"context"
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateDocument(db *mongo.Client, collection_name string, document any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	databaseName := config.Config.MongoDB.DB_Name
	dbCollection := db.Database(databaseName).Collection(collection_name)
	_, err := dbCollection.InsertOne(ctx, document)
	if err != nil {
		return err
	}
	return nil
}
