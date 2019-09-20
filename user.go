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

type UserData struct {
	Name  string `json:"name"`  // Name
	Grade uint   `json:"grade"` // Graduation Year
	Email string `json:"email"` // Email
	Late  uint   `json:"late"`  // Years Late
	Admin bool   `json:"admin"`
}

// Returns # of years they have been in school
func (u UserData) Years() uint {
	now := time.Now()

	if uint(now.Year()) >= u.Grade {
		return 4 - u.Late
	}

	years := now.Year() - (int(u.Grade) - 4) - int(u.Late)
	if now.Month() >= time.September {
		years += 1
	}
	return uint(years)
}

// Returns the grade of the user
func (u UserData) RealGrade() uint {
	now := time.Now()
	grade := uint(now.Year()) - u.Grade + 12
	if now.Month() >= time.September {
		grade += 1
	}
	return grade
}

// Returns the # of hours that the student should do
func (u UserData) Required() uint {
	return u.Years() * 20
}
