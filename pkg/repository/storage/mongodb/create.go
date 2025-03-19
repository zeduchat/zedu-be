package mongodb

import (
	"context"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
)

var _ = godotenv.Load()

var databaseName = os.Getenv("MONGO_DB_NAME")

func CreateEntry(db *mongo.Client, collection string, document interface{}) error {

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Second)
	defer cancel()

	dbCollection := db.Database(databaseName).Collection(collection) //replace with your database and collection names.
	_, err := dbCollection.InsertOne(ctx, document)
	if err != nil {
		return err
	}
	return nil
}
