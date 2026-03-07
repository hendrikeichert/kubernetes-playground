package database

import (
	"database/sql"
	"fmt"

	"github.com/hendrikeichert/gin-go/internal/config"
	"github.com/hendrikeichert/gin-go/internal/models"

	_ "github.com/lib/pq"
)

// PostgresDB holds the PostgreSQL connection
type PostgresDB struct {
	db *sql.DB
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg *config.PostgresConfig) (*PostgresDB, error) {
	db, err := sql.Open("postgres", cfg.JDBCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresDB{db: db}, nil
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// GetUsers retrieves all users from the database using the provided query
func (p *PostgresDB) GetUsers(query string) ([]models.User, error) {
	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return users, nil
}

// HealthCheck tests the database connection
func (p *PostgresDB) HealthCheck() error {
	return p.db.Ping()
}
