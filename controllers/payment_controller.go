package controllers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/AbaraEmmanuel/jaromind-backend/database"
	"github.com/AbaraEmmanuel/jaromind-backend/models"
)

func getPaymentsCollection() *mongo.Collection {
	return database.DB.Collection("payments")
}

// getUsersCollection matches LoginUser which reads from the "students" collection
func getUsersCollection() *mongo.Collection {
	return database.DB.Collection("students")
}

// ── PAYSTACK CONSTANTS ────────────────────────────────────────────────────────

func paystackSecretKey() string {
	return os.Getenv("PAYSTACK_SECRET_KEY")
}

const paystackBaseURL = "https://api.paystack.co"

// ================================================
// GET /user/trial-status?tutorId=xxx
// Returns whether the logged-in student has had a free trial
// with this specific tutor. Each tutor gets one free trial.
// ================================================

func GetTrialStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Not authenticated"})
		return
	}

	tutorID := c.Query("tutorId")
	if tutorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "tutorId query param is required"})
		return
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid user ID"})
		return
	}

	var result struct {
		TrialUsedWith []string `bson:"trialUsedWith"`
	}
	err = getUsersCollection().FindOne(ctx, bson.M{"_id": userObjectID}).Decode(&result)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "User not found"})
		return
	}

	// Check if this specific tutor's trial has been used
	alreadyUsed := false
	for _, tid := range result.TrialUsedWith {
		if tid == tutorID {
			alreadyUsed = true
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"canBookFree": !alreadyUsed,
		"tutorId":     tutorID,
	})
}

// ================================================
// POST /payments/initialize
// Called when student clicks "Pay & Book"
// Creates a Paystack payment and returns the payment URL
// ================================================

func InitializePayment(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Not authenticated"})
		return
	}

	var req struct {
		TutorID     string  `json:"tutorId" binding:"required"`
		Amount      float64 `json:"amount" binding:"required,gt=0"`
		CallbackURL string  `json:"callbackUrl" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	userObjectID, _ := primitive.ObjectIDFromHex(userID.(string))

	// Fetch user from "students" collection
	var user models.User
	err := getUsersCollection().FindOne(ctx, bson.M{"_id": userObjectID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "User not found"})
		return
	}

	// Fetch tutor to get their Calendly URL + rate
	var tutor models.TutorProfile
	err = getTutorsCollection().FindOne(ctx, bson.M{
		"$or": []bson.M{
			{"tutorId": req.TutorID},
			{"_id": convertToObjectID(req.TutorID)},
		},
		"isActive": true,
	}).Decode(&tutor)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Tutor not found"})
		return
	}

	reference := fmt.Sprintf("JARO-%s", uuid.New().String()[:8])
	amountKobo := int64(req.Amount * 100)

	payment := models.Payment{
		ID:           primitive.NewObjectID(),
		PaymentID:    uuid.New().String(),
		UserID:       userObjectID,
		TutorID:      req.TutorID,
		Amount:       req.Amount,
		Reference:    reference,
		Status:       "pending",
		CalendlyUrl:  tutor.CalendlyUrl,
		StudentName:  user.Name,
		StudentEmail: user.Email,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = getPaymentsCollection().InsertOne(ctx, payment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to create payment record"})
		return
	}

	paystackPayload := map[string]interface{}{
		"email":        user.Email,
		"amount":       amountKobo,
		"reference":    reference,
		"callback_url": req.CallbackURL,
		"metadata": map[string]interface{}{
			"tutorId":     req.TutorID,
			"userId":      userID,
			"studentName": user.Name,
			"paymentId":   payment.PaymentID,
		},
	}

	payloadBytes, _ := json.Marshal(paystackPayload)

	httpReq, _ := http.NewRequest("POST",
		paystackBaseURL+"/transaction/initialize",
		bytes.NewBuffer(payloadBytes),
	)
	httpReq.Header.Set("Authorization", "Bearer "+paystackSecretKey())
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to reach Paystack"})
		return
	}
	defer resp.Body.Close()

	var paystackResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&paystackResp)

	if paystackResp["status"] != true {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Paystack error: " + fmt.Sprintf("%v", paystackResp["message"]),
		})
		return
	}

	data := paystackResp["data"].(map[string]interface{})

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"paymentUrl": data["authorization_url"],
		"reference":  reference,
		"accessCode": data["access_code"],
	})
}

// ================================================
// POST /webhook/paystack
// ================================================

func PaystackWebhook(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot read body"})
		return
	}

	signature := c.GetHeader("x-paystack-signature")
	mac := hmac.New(sha512.New, []byte(paystackSecretKey()))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if signature != expected {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	if event["event"] != "charge.success" {
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	data := event["data"].(map[string]interface{})
	reference := data["reference"].(string)
	status := data["status"].(string)

	if status != "success" {
		c.JSON(http.StatusOK, gin.H{"status": "not success"})
		return
	}

	var payment models.Payment
	err = getPaymentsCollection().FindOne(ctx, bson.M{"reference": reference}).Decode(&payment)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	now := time.Now()
	getPaymentsCollection().UpdateOne(ctx,
		bson.M{"reference": reference},
		bson.M{"$set": bson.M{"status": "success", "updatedAt": now}},
	)

	// Record that this specific tutor's trial has been used
	getUsersCollection().UpdateOne(ctx,
		bson.M{"_id": payment.UserID},
		bson.M{
			"$addToSet": bson.M{"trialUsedWith": payment.TutorID},
			"$set":      bson.M{"updated_at": now},
		},
	)

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// ================================================
// GET /payments/verify/:reference
// ================================================

func VerifyPayment(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reference := c.Param("reference")

	httpReq, _ := http.NewRequest("GET",
		paystackBaseURL+"/transaction/verify/"+reference,
		nil,
	)
	httpReq.Header.Set("Authorization", "Bearer "+paystackSecretKey())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to verify with Paystack"})
		return
	}
	defer resp.Body.Close()

	var paystackResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&paystackResp)

	data := paystackResp["data"].(map[string]interface{})
	status := data["status"].(string)

	if status != "success" {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"success": false,
			"error":   "Payment not completed",
			"status":  status,
		})
		return
	}

	var payment models.Payment
	err = getPaymentsCollection().FindOne(ctx, bson.M{"reference": reference}).Decode(&payment)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Payment record not found"})
		return
	}

	now := time.Now()

	// Record that this specific tutor's trial has been used
	getUsersCollection().UpdateOne(ctx,
		bson.M{"_id": payment.UserID},
		bson.M{
			"$addToSet": bson.M{"trialUsedWith": payment.TutorID},
			"$set":      bson.M{"updated_at": now},
		},
	)

	getPaymentsCollection().UpdateOne(ctx,
		bson.M{"reference": reference},
		bson.M{"$set": bson.M{"status": "success", "updatedAt": now}},
	)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"status":      "success",
		"calendlyUrl": payment.CalendlyUrl,
		"tutorId":     payment.TutorID,
	})
}