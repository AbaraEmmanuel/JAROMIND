package main

import (
    "net/http" // ADD THIS IMPORT
    "os"

    "github.com/gin-gonic/gin"
    "github.com/AbaraEmmanuel/jaromind-backend/database"
    "github.com/AbaraEmmanuel/jaromind-backend/router"
)

func main() {
    // Initialize MongoDB
    database.InitDatabase()

    // Create ONE Gin router
    r := gin.Default()
    
    // CORS middleware - ADD THIS TO THE CORRECT ROUTER
    r.Use(func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(http.StatusNoContent)
            return
        }
        
        c.Next()
    })

    // Register routes on the SAME router
    router.RegisterRoutes(r)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    r.Run("0.0.0.0:" + port)
}