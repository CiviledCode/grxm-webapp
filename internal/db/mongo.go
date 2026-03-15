package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/civiledcode/grxm-webapp/internal/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoDB represents the MongoDB database instance.
var MongoDB *mongo.Database

// MongoClient represents the underlying MongoDB client.
var MongoClient *mongo.Client

// initMongo establishes a connection to MongoDB based on the provided configuration.
func initMongo(cfg *config.AppConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(cfg.MongoURI)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify the connection
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	log.Printf("Successfully connected to MongoDB provider at %s", cfg.MongoURI)

	MongoClient = client
	MongoDB = client.Database(cfg.MongoDB)

	return nil
}

// disconnectMongo gracefully closes the MongoDB connection.
func disconnectMongo(ctx context.Context) error {
	if MongoClient != nil {
		return MongoClient.Disconnect(ctx)
	}
	return nil
}
