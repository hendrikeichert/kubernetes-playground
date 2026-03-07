package database

import (
	"context"
	"fmt"

	"github.com/hendrikeichert/gin-go/internal/config"
	"github.com/hendrikeichert/gin-go/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB holds the MongoDB connection
type MongoDB struct {
	client *mongo.Client
	db     *mongo.Database
}

// NewMongoDB creates a new MongoDB connection
func NewMongoDB(cfg *config.MongoConfig) (*MongoDB, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.ConnectionString))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Verify connection
	if err := client.Ping(context.Background(), nil); err != nil {
		client.Disconnect(context.Background())
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &MongoDB{
		client: client,
		db:     client.Database(cfg.Database),
	}, nil
}

// Close closes the MongoDB connection
func (m *MongoDB) Close() error {
	return m.client.Disconnect(context.Background())
}

// GetUsers retrieves all users from MongoDB
func (m *MongoDB) GetUsers() ([]models.User, error) {
	collection := m.db.Collection("users")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer cursor.Close(context.Background())

	var users []models.User
	for cursor.Next(context.Background()) {
		var u models.User
		if err := cursor.Decode(&u); err != nil {
			return nil, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, u)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cursor: %w", err)
	}

	return users, nil
}

// HealthCheck tests the MongoDB connection
func (m *MongoDB) HealthCheck() error {
	return m.client.Ping(context.Background(), nil)
}
