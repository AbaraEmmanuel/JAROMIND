package main

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/okakafavour/jaromind-backend/config"
	"github.com/okakafavour/jaromind-backend/router"
)

func main() {

	// Initialize MongoDB
	config.InitDatabase()

	// Create Gin router
	r := gin.Default()
	router.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.Run("0.0.0.0:" + port)
}
