package mongodb

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectMongoDB(logger *utility.Logger, uri string) *mongo.Client {
	clientOptions := options.Client().ApplyURI(uri) // Replace with your MongoDB URI
	clientOptions.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to initialize MongoDB client: %v", err))
		fmt.Println("failed to connect to mongo DB")
		return nil
	}
	utility.LogAndPrint(logger, "Successfully connected to MongoDB")
	fmt.Println("connected to mongo DB  ✅ ")
	storage.DB.Mongo = client
	return client
}
