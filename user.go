package main

import (
	"encoding/csv"
	"time"
)

const (
	StudentListPath = "./data/students.csv"
	AdminListPath   = "./data/admins.txt"
)

var _ = csv.NewReader

type User struct {
	Name  string `json:"name"`  // Name
	Grade uint   `json:"grade"` // Graduation Year
	Email string `json:"email"` // Email
	Late  uint   `json:"late"`  // Years Late
	Admin bool   `json:"admin"`
}

// Returns the grade of the user
func (u User) RealGrade() uint {
	now := time.Now()
	grade := uint(now.Year()) + 12 - u.Grade
	if now.Month() >= time.September {
		grade += 1
	}
	return grade
}

// Returns the # of hours that the student should do
func (u User) Required() uint {
	return (u.RealGrade() - 8) * 20
}
