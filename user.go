package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
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
	if u.Grade == 0 {
		return 0
	}

	grade := uint(t.Year()) + 12 - u.Grade
	if t.Month() >= time.July {
		grade += 1
	}
	return grade
}

// Method Required returns the # of hours that the student should do
func (u User) Required() uint {
	return (u.GradeNow() - 8 - u.Late) * 20
}

func UsersFromCSV(r io.Reader) ([]User, error) {
	records, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) != 0 && len(records[0]) != 0 && strings.EqualFold(records[0][0], "Name") {
		records = records[1:]
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no users found")
	}

	if len(records[0]) != 4 {
		return nil, fmt.Errorf("wrong number of fields")
	}

	out := []User(nil)
	for _, record := range records {
		user := User{}
		user.Name = record[0]
		grade, err := strconv.ParseUint(record[1], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid graduation year: '%v'", record[1])
		}
		user.Grade = uint(grade)
		user.Email = record[2]
		late, err := strconv.ParseUint(record[3], 10, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid # of years late: '%v'", record[3])
		}
		user.Late = uint(late)
		out = append(out, user)
	}
	return out, nil
}
