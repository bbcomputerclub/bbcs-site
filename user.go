package main

import (
	"encoding/csv"
	"time"
)

var _ = csv.NewReader

type User struct {
	Name  string `json:"name"`  // Name
	Grade uint   `json:"grade"` // Graduation Year
	Email string `json:"email"` // Email
	Late  uint   `json:"late"`  // Years Late
	Admin bool   `json:"admin"` // User type: true for admin, false for student
}

// Method RealGrade returns the grade of the user
func (u User) RealGrade() uint {
	now := time.Now()
	grade := uint(now.Year()) + 12 - u.Grade
	if now.Month() >= time.September {
		grade += 1
	}
	return grade
}

// Method Required returns the # of hours that the student should do
func (u User) Required() uint {
	return (u.RealGrade() - 8) * 20
}
