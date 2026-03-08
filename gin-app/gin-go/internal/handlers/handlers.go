package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hendrikeichert/gin-go/internal/database"
)

// SetupRoutes configures the Gin router with handlers
func SetupRoutes(r *gin.Engine, db database.Database) {
	r.GET("/users", getUsers(db))
	r.GET("/health", healthCheck(db))
}

// getUsers creates a handler for retrieving users
func getUsers(db database.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := db.GetUsers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, users)
	}
}

// healthCheck creates a handler for testing database connectivity
func healthCheck(db database.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbName := c.Query("db")
		if dbName == "" {
			dbName = "all" // Default to checking all databases
		}

		err := db.HealthCheck(dbName)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  err.Error(),
				"db":     dbName,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"db":     dbName,
		})
	}
}
