package config

import (
	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	AppPort  string         `mapstructure:"QUARKUS_HTTP_PORT"`
	AppEnv   string         `mapstructure:"QUARKUS_PROFILE"`
	Postgres PostgresConfig `mapstructure:"QUARKUS_DATASOURCE"`
	Mongo    MongoConfig    `mapstructure:"QUARKUS_MONGODB"`
}

// PostgresConfig holds PostgreSQL-specific configuration
type PostgresConfig struct {
	JDBCURL  string `mapstructure:"JDBC_URL"`
	Username string `mapstructure:"USERNAME"`
	Password string `mapstructure:"PASSWORD"`
	Query    string `mapstructure:"QUERY"`
}

// MongoConfig holds MongoDB-specific configuration
type MongoConfig struct {
	ConnectionString string `mapstructure:"CONNECTION_STRING"`
	Database         string `mapstructure:"DATABASE"`
}

// LoadConfig loads configuration from .env file or environment variables
func LoadConfig() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv() // Bind to environment variables

	// Set defaults
	viper.SetDefault("QUARKUS_HTTP_PORT", "8080")
	viper.SetDefault("QUARKUS_PROFILE", "dev")
	viper.SetDefault("QUARKUS_DATASOURCE_QUERY", "SELECT id, name, email FROM users")

	// Read .env file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// .env file not found, rely on environment variables
		} else {
			return nil, err
		}
	}

	// Unmarshal into struct
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
