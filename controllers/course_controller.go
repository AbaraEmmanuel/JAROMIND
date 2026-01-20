package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/AbaraEmmanuel/jaromind-backend/database"
	"github.com/AbaraEmmanuel/jaromind-backend/models"
)

// Helper functions to get collections
func getCoursesCollection() *mongo.Collection {
	return database.DB.Collection("courses")
}

func getEnrollmentsCollection() *mongo.Collection {
	return database.DB.Collection("enrollments")
}

func getReviewsCollection() *mongo.Collection {
	return database.DB.Collection("reviews")
}

// GetAllCourses - Get all active courses with optional filters
func GetAllCourses(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build filter
	filter := bson.M{"isActive": true}

	// Optional filters
	if courseType := c.Query("type"); courseType != "" {
		filter["type"] = courseType
	}
	if classLevel := c.Query("classLevel"); classLevel != "" {
		filter["classLevel"] = classLevel
	}
	if subject := c.Query("subject"); subject != "" {
		filter["subject"] = subject
	}
	if status := c.Query("status"); status != "" {
		filter["status"] = status
	}
	if category := c.Query("category"); category != "" {
		filter["category"] = category
	}
	if featured := c.Query("featured"); featured == "true" {
		filter["isFeatured"] = true
	}

	// Sorting
	sortBy := c.DefaultQuery("sortBy", "createdAt")
	order := c.DefaultQuery("order", "desc")
	sortOrder := 1
	if order == "desc" {
		sortOrder = -1
	}

	opts := options.Find().SetSort(bson.D{{Key: sortBy, Value: sortOrder}})

	cursor, err := getCoursesCollection().Find(ctx, filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch courses"})
		return
	}
	defer cursor.Close(ctx)

	var courses []models.Course
	if err = cursor.All(ctx, &courses); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse courses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"courses": courses,
		"count":   len(courses),
	})
}

// GetCourseByID - Get single course with full details
func GetCourseByID(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	courseID := c.Param("id")
	
	var course models.Course
	err := getCoursesCollection().FindOne(ctx, bson.M{"id": courseID, "isActive": true}).Decode(&course)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Get reviews for this course
	cursor, _ := getReviewsCollection().Find(ctx, bson.M{"courseId": courseID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(10))
	var reviews []models.Review
	cursor.All(ctx, &reviews)

	c.JSON(http.StatusOK, gin.H{
		"course":  course,
		"reviews": reviews,
	})
}

// CreateCourse - Admin only
func CreateCourse(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var course models.Course
	
	if err := c.ShouldBindJSON(&course); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	course.ID = uuid.New().String()
	course.CreatedAt = time.Now()
	course.UpdatedAt = time.Now()
	course.IsActive = true

	_, err := getCoursesCollection().InsertOne(ctx, course)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create course"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Course created successfully",
		"course":  course,
	})
}

// UpdateCourse - Admin only
func UpdateCourse(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	courseID := c.Param("id")

	var updates models.Course
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates.UpdatedAt = time.Now()
	
	// Convert to bson.M for update
	updateDoc := bson.M{"$set": updates}

	result, err := getCoursesCollection().UpdateOne(ctx, bson.M{"id": courseID}, updateDoc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update course"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Course updated successfully"})
}

// DeleteCourse - Soft delete
func DeleteCourse(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	courseID := c.Param("id")
	
	_, err := getCoursesCollection().UpdateOne(ctx, bson.M{"id": courseID}, bson.M{"$set": bson.M{"isActive": false}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete course"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Course deleted successfully"})
}

// EnrollInCourse - Student enrollment
func EnrollInCourse(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	courseID := c.Param("courseId")
	userID, exists := c.Get("userID")
	
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if course exists
	var course models.Course
	err := getCoursesCollection().FindOne(ctx, bson.M{"id": courseID, "isActive": true}).Decode(&course)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Check if already enrolled
	count, _ := getEnrollmentsCollection().CountDocuments(ctx, bson.M{"userId": userID, "courseId": courseID})
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Already enrolled in this course"})
		return
	}

	// Create enrollment
	enrollment := models.Enrollment{
		ID:               uuid.New().String(),
		UserID:           userID.(string),
		CourseID:         courseID,
		EnrolledAt:       time.Now(),
		LastAccessedAt:   time.Now(),
		Progress:         0,
		CompletedLessons: []string{},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	_, err = getEnrollmentsCollection().InsertOne(ctx, enrollment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enroll in course"})
		return
	}

	// Increment enrollment count
	getCoursesCollection().UpdateOne(ctx, bson.M{"id": courseID}, bson.M{"$inc": bson.M{"enrollmentCount": 1}})

	c.JSON(http.StatusOK, gin.H{
		"message":    "Successfully enrolled",
		"enrollment": enrollment,
	})
}

// GetUserEnrollments - Get all courses user is enrolled in
func GetUserEnrollments(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, exists := c.Get("userID")
	
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	cursor, err := getEnrollmentsCollection().Find(ctx, bson.M{"userId": userID}, options.Find().SetSort(bson.D{{Key: "lastAccessedAt", Value: -1}}))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch enrollments"})
		return
	}
	defer cursor.Close(ctx)

	var enrollments []models.Enrollment
	cursor.All(ctx, &enrollments)

	// Get course details for each enrollment
	var enrollmentDetails []gin.H
	for _, enrollment := range enrollments {
		var course models.Course
		err := getCoursesCollection().FindOne(ctx, bson.M{"id": enrollment.CourseID}).Decode(&course)
		if err == nil {
			enrollmentDetails = append(enrollmentDetails, gin.H{
				"enrollment": enrollment,
				"course":     course,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"enrollments": enrollmentDetails,
		"count":       len(enrollmentDetails),
	})
}

// UpdateProgress - Update student's course progress
func UpdateProgress(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	courseID := c.Param("courseId")
	userID, exists := c.Get("userID")
	
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request struct {
		Progress         int      `json:"progress"`
		CompletedLessons []string `json:"completedLessons"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := bson.M{
		"$set": bson.M{
			"progress":         request.Progress,
			"completedLessons": request.CompletedLessons,
			"lastAccessedAt":   time.Now(),
			"updatedAt":        time.Now(),
		},
	}

	// If course is 100% complete
	if request.Progress >= 100 {
		update["$set"].(bson.M)["completedAt"] = time.Now()
	}

	_, err := getEnrollmentsCollection().UpdateOne(ctx, bson.M{"userId": userID, "courseId": courseID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Progress updated successfully"})
}

// AddReview - Add a course review
func AddReview(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	courseID := c.Param("courseId")
	userID, exists := c.Get("userID")
	
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if user is enrolled
	count, _ := getEnrollmentsCollection().CountDocuments(ctx, bson.M{"userId": userID, "courseId": courseID})
	if count == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Must be enrolled to review"})
		return
	}

	var request struct {
		Rating  int    `json:"rating" binding:"required,min=1,max=5"`
		Comment string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userName := c.GetString("userName")
	userAvatar := c.GetString("userAvatar")

	review := models.Review{
		ID:         uuid.New().String(),
		CourseID:   courseID,
		UserID:     userID.(string),
		Rating:     request.Rating,
		Comment:    request.Comment,
		UserName:   userName,
		UserAvatar: userAvatar,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	_, err := getReviewsCollection().InsertOne(ctx, review)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add review"})
		return
	}

	// Update course rating
	updateCourseRating(courseID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Review added successfully",
		"review":  review,
	})
}

// Helper function to recalculate course rating
func updateCourseRating(courseID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, _ := getReviewsCollection().Find(ctx, bson.M{"courseId": courseID})
	var reviews []models.Review
	cursor.All(ctx, &reviews)

	if len(reviews) == 0 {
		return
	}

	var totalRating int
	for _, review := range reviews {
		totalRating += review.Rating
	}

	avgRating := float64(totalRating) / float64(len(reviews))
	
	getCoursesCollection().UpdateOne(ctx, bson.M{"id": courseID}, bson.M{
		"$set": bson.M{
			"rating":      avgRating,
			"reviewCount": len(reviews),
		},
	})
}

// GetCourseStats - Get statistics for a course
func GetCourseStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	courseID := c.Param("id")

	var course models.Course
	err := getCoursesCollection().FindOne(ctx, bson.M{"id": courseID}).Decode(&course)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	completionCount, _ := getEnrollmentsCollection().CountDocuments(ctx, bson.M{
		"courseId": courseID,
		"completedAt": bson.M{"$ne": nil},
	})

	completionRate := 0.0
	if course.EnrollmentCount > 0 {
		completionRate = float64(completionCount) / float64(course.EnrollmentCount) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"enrollments":    course.EnrollmentCount,
		"completions":    completionCount,
		"rating":         course.Rating,
		"reviewCount":    course.ReviewCount,
		"completionRate": completionRate,
	})
}