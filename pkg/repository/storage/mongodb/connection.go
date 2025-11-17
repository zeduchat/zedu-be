package mongodb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStore struct {
	Client *mongo.Client
	Lock   sync.RWMutex
}

func (m *MongoStore) GetClient() (*mongo.Client, error) {
	m.Lock.RLock()
	defer m.Lock.RUnlock()

	if m.Client == nil {
		return nil, fmt.Errorf("MongoDB Client is not initialized")
	}

	return m.Client, nil
}

func (m *MongoStore) SetClient(client *mongo.Client) {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	m.Client = client
}

func (m *MongoStore) IsClientAvailable() bool {
	m.Lock.RLock()
	defer m.Lock.RUnlock()

	return storage.DB.Mongo != nil
}

func ConnectMongoDB(logger *utility.Logger, MongoConfig config.MongoDB, store *MongoStore) {
	clientOptions := options.Client().ApplyURI(MongoConfig.Mongo_URI)
	clientOptions.SetConnectTimeout(10 * time.Second)

	retryDelay := 5 * time.Second

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		logger.Info("🔄🔄🔄 Attempting to connect to MongoDB...")
		fmt.Println("🔄🔄🔄 Attempting to connect to MongoDB...")
		client, err := mongo.Connect(ctx, clientOptions)
		if err != nil {
			// logger.Error("❌❌❌ Failed to initialize MongoDB client: %v ❌❌❌", err)
			// fmt.Printf("❌❌❌ Failed to initialize MongoDB client: %v ❌❌❌\n", err)
		} else {
			err = client.Ping(ctx, nil)
			if err != nil {
				logger.Error("❌❌❌ Failed to ping MongoDB: %v ❌❌❌", err)
				fmt.Printf("❌❌❌ Failed to ping MongoDB: %v ❌❌❌\n", err)
				client.Disconnect(ctx)
			} else {
				logger.Info("✅✅✅ Successfully connected and pinged MongoDB ✅✅✅")
				fmt.Println("✅✅✅ Successfully connected and pinged MongoDB ✅✅✅")
				store.SetClient(client)
				storage.DB.Mongo = client // Maintain backward compatibility
				EnsureAgentCollectionsIndex(logger, client)
				return
			}
		}

		logger.Info("⏳ Retrying MongoDB connection in %v", retryDelay)
		fmt.Printf("⏳ Retrying MongoDB connection in %v\n", retryDelay)
		time.Sleep(retryDelay)
		retryDelay = min(retryDelay*2, 60*time.Second) // Exponential backoff, max 60s
	}
}

func MonitorMongoConnection(logger *utility.Logger, MongoConfig config.MongoDB, store *MongoStore) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		client, err := store.GetClient()
		if err != nil || client == nil {
			logger.Info("⚠️⚠️⚠️ MongoDB client unavailable, attempting to reconnect....")
			fmt.Println("⚠️⚠️⚠️ MongoDB client unavailable, attempting to reconnect....")
			ConnectMongoDB(logger, MongoConfig, store)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = client.Ping(ctx, nil)
		cancel()

		if err != nil {
			logger.Error("❌❌❌ MongoDB ping failed %v ❌❌❌", err)
			fmt.Printf("❌❌❌ MongoDB ping failed %v ❌❌❌\n", err)
			store.SetClient(nil)
			ConnectMongoDB(logger, MongoConfig, store)
		} else {
			// fmt.Println("✅✅✅ MongoDB connection is healthy ✅✅✅")
		}
	}
}

func StartMongoDBConnection(logger *utility.Logger, MongoConfig config.MongoDB) *MongoStore {
	store := &MongoStore{}

	go MonitorMongoConnection(logger, MongoConfig, store)
	go ConnectMongoDB(logger, MongoConfig, store)
	return store
}

//ensures db level integrity for the creation of unique collections based on agent_id
func EnsureAgentCollectionsIndex(logger *utility.Logger, db *mongo.Client) error {
	databaseName := config.Config.MongoDB.DB_Name
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	database := db.Database(databaseName)

	agentCollections := database.Collection("agent_collections")
	_, err := agentCollections.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"agent_id": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create unique index on agent_id: %v", err)
	}

	fmt.Println("✅✅✅ Unique index on agent_id created successfully ✅✅✅")

	return nil
}
