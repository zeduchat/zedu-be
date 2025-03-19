package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/utility"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectMongoDB(logger *utility.Logger, uri string) *mongo.Client {
	clientOptions := options.Client().ApplyURI(uri) // Replace with your MongoDB URI
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to initialize MongoDB client: %v", err))
		fmt.Println("failed to connect to mongo DB")
		return nil
	}
	utility.LogAndPrint(logger, "Successfully connected to MongoDB")
	fmt.Println("connected to mongo DB  ✅ ")
	return client
}
