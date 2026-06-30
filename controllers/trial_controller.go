// controllers/trial_controller.go
package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// resolveUserObjectID extracts "userID" from the Gin context (set by
// JWTAuthMiddleware) and converts it to a primitive.ObjectID.
func resolveUserObjectID(c *gin.Context) (primitive.ObjectID, bool) {
	raw, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
		return primitive.NilObjectID, false
	}

	hexID, ok := raw.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Invalid user ID type in token"})
		return primitive.NilObjectID, false
	}

	oid, err := primitive.ObjectIDFromHex(hexID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Malformed user ID in token"})
		return primitive.NilObjectID, false
	}

	return oid, true
}

// ──────────────────────────────────────────────────────────────
// POST /user/trial-status/consume
// Body: { "tutorId": "xxx" }
//
// Appends tutorId to the user's trialUsedWith array.
// Each tutor gets one free trial per student — not one global trial.
// Idempotent: MongoDB $addToSet means calling it twice is safe.
// ──────────────────────────────────────────────────────────────
func ConsumeTrialStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userOID, ok := resolveUserObjectID(c)
	if !ok {
		return
	}

	var req struct {
		TutorID string `json:"tutorId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "tutorId is required in request body",
		})
		return
	}

	// Verify user exists
	count, err := getUsersCollection().CountDocuments(ctx, bson.M{"_id": userOID})
	if err != nil || count == 0 {
		if err == mongo.ErrNoDocuments || count == 0 {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch user"})
		return
	}

	// $addToSet is idempotent — adds tutorId only if not already in the array
	_, err = getUsersCollection().UpdateOne(
		ctx,
		bson.M{"_id": userOID},
		bson.M{
			"$addToSet": bson.M{"trialUsedWith": req.TutorID},
			"$set":      bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update trial status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trial recorded for tutor " + req.TutorID,
		"tutorId": req.TutorID,
	})
}