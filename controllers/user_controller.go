package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/okakafavour/jaromind-backend/models"

	// "github.com/okakafavour/jaromind-backend/services"
	servicesimpl "github.com/okakafavour/jaromind-backend/services_impl"
)

// var userService services.UserService

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

func RegisterUser(c *gin.Context) {
	var request RegisterRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	student := models.User{
		Name:     request.Name,
		Email:    request.Email,
		Password: request.Password,
	}

	err := servicesimpl.NewUserService().Register(student)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Registration successful! Please check your email to verify your account.",
	})
}

type LoginRequest struct {
	Email    string `bson:"email" binding:"required,email"`
	Password string `bson:"password" binding:"required"`
}

func LoginUser(c *gin.Context) {
	var request LoginRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	token, err := servicesimpl.NewUserService().Login(request.Email, request.Password)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "Login successfully",
		"token":   token,
	})
}

func GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	email, _ := c.Get("email")
	role, _ := c.Get("role")

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"email":   email,
		"role":    role,
	})
}
