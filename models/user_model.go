package models

import (
"time"
"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
Name      string             `bson:"name" json:"name"`
Email     string             `bson:"email" json:"email"`
Phone     string             `bson:"phone" json:"phone"`
Level     string             `bson:"level" json:"level"`
Password  string             `bson:"password" json:"-"`
Code      string             `bson:"code,omitempty" json:"code,omitempty"`
Verified  bool               `bson:"verified" json:"verified"`
TrialUsedWith []string 		`bson:"trialUsedWith" json:"trialUsedWith"`
CreatedAt time.Time          `bson:"created_at" json:"created_at"`
UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// Add this Admin struct - keep it separate from User
type Admin struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Email     string             `bson:"email" json:"email"`
    Password  string             `bson:"password" json:"password"`
    Name      string             `json:"name" bson:"name"`
    CreatedAt primitive.DateTime `bson:"createdAt" json:"createdAt"`
    IsActive  bool               `bson:"isActive" json:"isActive"`
}

// Payment tracks a Paystack transaction for a tutoring session
type Payment struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PaymentID     string             `bson:"paymentId" json:"paymentId"`         // UUID
	UserID        primitive.ObjectID `bson:"userId" json:"userId"`
	TutorID       string             `bson:"tutorId" json:"tutorId"`
	Amount        float64            `bson:"amount" json:"amount"`               // in NGN kobo (smallest unit)
	Reference     string             `bson:"reference" json:"reference"`         // Paystack reference
	Status        string             `bson:"status" json:"status"`               // "pending" | "success" | "failed"
	CalendlyUrl   string             `bson:"calendlyUrl" json:"calendlyUrl"`     // unlocked after payment
	StudentName   string             `bson:"studentName" json:"studentName"`
	StudentEmail  string             `bson:"studentEmail" json:"studentEmail"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt" json:"updatedAt"`
}