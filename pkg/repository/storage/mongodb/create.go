package mongodb

import (
	"context"
	"strings"
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateEntry(db *mongo.Client, collection string, document interface{}) error {
	databaseName := config.Config.MongoDB.DB_Name
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dbCollection := db.Database(databaseName).Collection(collection)
	_, err := dbCollection.InsertOne(ctx, document)
	if err != nil {
		return err
	}
	return nil
}

func CreateCollection(db *mongo.Client, collection string) error {
	databaseName := config.Config.MongoDB.DB_Name
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := db.Database(databaseName).CreateCollection(ctx, collection)
	if err != nil {
		return err
	}
	return nil
}

func ListCollections(db *mongo.Client, prefix string) ([]string, error) {
	databaseName := config.Config.MongoDB.DB_Name
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cursor, err := db.Database(databaseName).ListCollections(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	var filteredCollections []string

	for cursor.Next(ctx) {
		var collection bson.M
		if err := cursor.Decode(&collection); err != nil {
			return nil, err
		}

		if name, ok := collection["name"].(string); ok && strings.HasPrefix(name, prefix) {
			name = utility.ExtractCollectionName(name)
			filteredCollections = append(filteredCollections, name)
		}
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return filteredCollections, nil
}
