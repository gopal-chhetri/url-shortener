package main

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize default Gin router (includes Logging and Recovery middleware)
	r := gin.Default()

	// Define a simple GET route
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// Start server on 0.0.0.0:8080 by default
	r.Run() 
}

