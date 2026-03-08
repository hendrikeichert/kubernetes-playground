package main

import (
	"fmt"
	"log"

	"github.com/hendrikeichert/gin-go/internal/config"
	"github.com/hendrikeichert/gin-go/internal/database"
	"github.com/hendrikeichert/gin-go/internal/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	if db != nil {
		log.Println("Database connection established successfully")
		defer db.Close()
	} else {
		log.Println("No database connection established")
	}

	// Set up Gin router
	r := gin.Default()

	// Initialize handlers
	handlers.SetupRoutes(r, db)

	// Start server
	port := cfg.AppPort
	if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
