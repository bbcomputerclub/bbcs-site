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

// Method GradeNow returns the grade of the user
func (u User) GradeNow() uint {
	return u.GradeAt(time.Now())
}

// Method GradeAt returns what grade the user was in at a given instant
func (u User) GradeAt(t time.Time) uint {
	grade := uint(t.Year()) + 12 - u.Grade
	if t.Month() >= time.July {
		grade += 1
	}
	return grade
}

// Method Required returns the # of hours that the student should do
func (u User) Required() uint {
	return (u.GradeNow() - 8) * 20
}
