package router

import (
	"time"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/AbaraEmmanuel/jaromind-backend/controllers"
	"github.com/AbaraEmmanuel/jaromind-backend/middleware"
)

func RegisterRoutes(router *gin.Engine) {
	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8000", "http://127.0.0.1:8000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Public routes
	router.POST("/register", controllers.RegisterUser)
	router.POST("/login", controllers.LoginUser)
	
	// NEW: Course routes (public - anyone can view courses)
	router.GET("/courses", controllers.GetAllCourses)
	router.GET("/courses/:id", controllers.GetCourseByID)

	// Protected routes
	protected := router.Group("/user")
	protected.Use(middleware.JWTAuthMiddleware())
	{
		protected.GET("/profile", controllers.GetProfile)
		protected.POST("/enroll/:courseId", controllers.EnrollInCourse)
		protected.GET("/enrollments", controllers.GetUserEnrollments)
	}
}