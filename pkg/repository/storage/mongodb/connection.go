package mongodb

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectMongoDB(logger *utility.Logger, MongoConfig config.MongoDB) *mongo.Client {
    clientOptions := options.Client().ApplyURI(MongoConfig.Mongo_URI)
    clientOptions.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})

    var client *mongo.Client
    var err error

    maxRetries := 10
    retryDelay := 5 * time.Second

    for i := 0; i < maxRetries; i++ {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        logger.Info("Attempting to connect to MongoDB (attempt %d)...", i+1)
        client, err = mongo.Connect(ctx, clientOptions)
        if err != nil {
            logger.Error("Failed to initialize MongoDB client: %v", err)
            fmt.Println("failed to connect to mongo DB ❌❌❌❌❌")
        } else {

            // Verify connection with Ping
            err = client.Ping(ctx, nil)
            if err != nil {
                logger.Error("Failed to ping MongoDB: %v", err)
                fmt.Println("failed to ping mongo DB ❌❌❌❌❌")
                client.Disconnect(ctx)
            } else {
                utility.LogAndPrint(logger, "Successfully connected and pinged MongoDB")
                fmt.Println("connected to mongo DB ✅✅✅✅✅✅✅✅")
                storage.DB.Mongo = client
                return client
            }

        }

        // Wait before retrying
        time.Sleep(retryDelay)
        retryDelay *= 2 // Exponential backoff
    }

    logger.Error("Exceeded maximum retries to connect to MongoDB")
    return nil
}