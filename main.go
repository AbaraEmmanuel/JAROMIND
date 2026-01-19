package main

import (
        "os"

        "github.com/gin-gonic/gin"
        "github.com/AbaraEmmanuel/jaromind-backend/database"
        "github.com/AbaraEmmanuel/jaromind-backend/router"
)

func main() {

        // Initialize MongoDB
        database.InitDatabase()

        // Create Gin router
        r := gin.Default()
        router.RegisterRoutes(r)

        port := os.Getenv("PORT")
        if port == "" {
                port = "8080"
        }

        r.Run("0.0.0.0:" + port)
}
