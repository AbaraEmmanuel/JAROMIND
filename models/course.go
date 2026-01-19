package models

import (
"time"
)

type Course struct {
ID          string    `bson:"_id,omitempty" json:"id"`
Title       string    `bson:"title" json:"title"`
Description string    `bson:"description" json:"description"`
// Add other fields as needed
IsActive    bool      `bson:"is_active" json:"is_active"`
CreatedAt   time.Time `bson:"created_at" json:"created_at"`
UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}
