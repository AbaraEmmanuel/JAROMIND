package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/AbaraEmmanuel/jaromind-backend/database"
	"github.com/AbaraEmmanuel/jaromind-backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetAllCourses(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get courses collection
	collection := database.GetCollection("courses")
	
	// Find active courses
	cursor, err := collection.Find(ctx, bson.M{"is_active": true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch courses"})
		return
	}
	defer cursor.Close(ctx)

	var courses []models.Course
	if err = cursor.All(ctx, &courses); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode courses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"courses": courses,
		"count":   len(courses),
	})
}

func GetCourseByID(c *gin.Context) {
	courseID := c.Param("id")
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := database.GetCollection("courses")
	
	// Convert string ID to MongoDB ObjectID
	objID, err := primitive.ObjectIDFromHex(courseID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID format"})
		return
	}

	var course models.Course
	err = collection.FindOne(ctx, bson.M{
		"_id":       objID,
		"is_active": true,
	}).Decode(&course)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch course"})
		}
		return
	}

	c.JSON(http.StatusOK, course)
}

func EnrollInCourse(c *gin.Context) {
	courseID := c.Param("courseId")
	userID, exists := c.Get("userID")
	
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if course exists
	coursesCollection := database.GetCollection("courses")
	objID, err := primitive.ObjectIDFromHex(courseID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}

	var course models.Course
	err = coursesCollection.FindOne(ctx, bson.M{
		"_id":       objID,
		"is_active": true,
	}).Decode(&course)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch course"})
		}
		return
	}

	// Check if already enrolled
	enrollmentsCollection := database.GetCollection("enrollments")
	var existingEnrollment models.Enrollment
	err = enrollmentsCollection.FindOne(ctx, bson.M{
		"user_id":   userID,
		"course_id": courseID,
	}).Decode(&existingEnrollment)

	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Already enrolled in this course"})
		return
	}

	// Create enrollment - using UUID for ID (or you can use primitive.NewObjectID())
	enrollment := models.Enrollment{
		ID:               uuid.New().String(), // Use string ID or primitive.ObjectID
		UserID:           userID.(string),
		CourseID:         courseID,
		EnrolledAt:       time.Now(),
		Progress:         0,
		CompletedLessons: []string{},
		CreatedAt:        time.Now(),
	}

	_, err = enrollmentsCollection.InsertOne(ctx, enrollment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enroll in course"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Successfully enrolled",
		"enrollment": enrollment,
	})
}

func GetUserEnrollments(c *gin.Context) {
	userID, exists := c.Get("userID")
	
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := database.GetCollection("enrollments")
	cursor, err := collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch enrollments"})
		return
	}
	defer cursor.Close(ctx)

	var enrollments []models.Enrollment
	if err = cursor.All(ctx, &enrollments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode enrollments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enrollments": enrollments,
		"count":       len(enrollments),
	})
}