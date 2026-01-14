package router

import (
	"time"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/okakafavour/jaromind-backend/controllers"
	"github.com/okakafavour/jaromind-backend/middleware"
)

func RegisterRoutes(router *gin.Engine) {
	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8000", "http://127.0.0.1:8000"}, // Specific origins for development
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length", "Authorization"},
		AllowCredentials: true, // Set to true if you're using cookies/sessions
		MaxAge:           12 * time.Hour,
	}))

	// Public routes
	router.POST("/register", controllers.RegisterUser)
	router.POST("/login", controllers.LoginUser)

	// Protected routes
	protected := router.Group("/user")
	protected.Use(middleware.JWTAuthMiddleware())
	{
		protected.GET("/profile", controllers.GetProfile)
	}
}
