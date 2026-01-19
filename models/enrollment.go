package models

import (
"time"
)

type Enrollment struct {
ID               string    `bson:"_id,omitempty" json:"id"`
UserID           string    `bson:"user_id" json:"user_id"`
CourseID         string    `bson:"course_id" json:"course_id"`
EnrolledAt       time.Time `bson:"enrolled_at" json:"enrolled_at"`
Progress         int       `bson:"progress" json:"progress"`
CompletedLessons []string  `bson:"completed_lessons" json:"completed_lessons"`
CreatedAt        time.Time `bson:"created_at" json:"created_at"`
}
