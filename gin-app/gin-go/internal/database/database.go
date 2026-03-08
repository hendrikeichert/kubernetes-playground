package database

import (
	"fmt"

	"github.com/hendrikeichert/gin-go/internal/config"
	"github.com/hendrikeichert/gin-go/internal/models"
)

// Database defines the interface for database operations
type Database interface {
	Close() error
	GetUsers() ([]models.User, error)
	HealthCheck(string) error
}

// MultiDatabase holds multiple database connections
type MultiDatabase struct {
	Postgres *PostgresDB
	Mongo    *MongoDB
	Config   *config.Config // Store config for query access
}

// NewDatabase initializes all configured databases
func NewDatabase(cfg *config.Config) (*MultiDatabase, error) {
	// Initialize PostgreSQL
	postgresDB, err := NewPostgresDB(&cfg.Postgres)
	if err != nil {
		postgresDB.Close() // Clean up on failure
		return nil, fmt.Errorf("failed to initialize PostgreSQL: %w", err)
	}

	// Initialize MongoDB
	mongoDB, err := NewMongoDB(&cfg.Mongo)
	if err != nil {
		mongoDB.Close() // Clean up on failure
		return nil, fmt.Errorf("failed to initialize MongoDB: %w", err)
	}

	return &MultiDatabase{
		Postgres: postgresDB,
		Mongo:    mongoDB,
		Config:   cfg,
	}, nil
}

// Close closes all database connections
func (m *MultiDatabase) Close() error {
	var errs []error
	if err := m.Postgres.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := m.Mongo.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing databases: %v", errs)
	}
	return nil
}

// GetUsers retrieves users from all databases
func (m *MultiDatabase) GetUsers() ([]models.User, error) {
	// Run queries in parallel
	type result struct {
		users []models.User
		err   error
	}
	postgresChan := make(chan result)
	mongoChan := make(chan result)

	go func() {
		users, err := m.Postgres.GetUsers(m.Config.Postgres.Query)
		postgresChan <- result{users, err}
	}()
	go func() {
		users, err := m.Mongo.GetUsers()
		mongoChan <- result{users, err}
	}()

	var users []models.User
	for i := 0; i < 2; i++ {
		select {
		case res := <-postgresChan:
			if res.err != nil {
				return nil, fmt.Errorf("failed to get users from Postgres: %w", res.err)
			}
			users = append(users, res.users...)
		case res := <-mongoChan:
			if res.err != nil {
				return nil, fmt.Errorf("failed to get users from MongoDB: %w", res.err)
			}
			users = append(users, res.users...)
		}
	}

	// Deduplicate users by ID
	userMap := make(map[int]models.User)
	for _, u := range users {
		userMap[u.ID] = u
	}
	dedupedUsers := make([]models.User, 0, len(userMap))
	for _, u := range userMap {
		dedupedUsers = append(dedupedUsers, u)
	}

	return dedupedUsers, nil
}

// HealthCheck tests connectivity to the specified database
func (m *MultiDatabase) HealthCheck(dbName string) error {
	switch dbName {
	case "postgres":
		return m.Postgres.HealthCheck()
	case "mongo":
		return m.Mongo.HealthCheck()
	case "all":
		var errs []error
		if err := m.Postgres.HealthCheck(); err != nil {
			errs = append(errs, fmt.Errorf("postgres: %w", err))
		}
		if err := m.Mongo.HealthCheck(); err != nil {
			errs = append(errs, fmt.Errorf("mongo: %w", err))
		}
		if len(errs) > 0 {
			return fmt.Errorf("health check errors: %v", errs)
		}
		return nil
	default:
		return fmt.Errorf("invalid database name: %s", dbName)
	}
}
