package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	StudentListPath = "./data/students.csv"
	AdminListPath   = "./data/admins.txt"
)

type UserData struct {
	Name  string // Name
	Grade uint   // Graduation Year
	Email string // Email
	Late  uint   // Years Late
	//	Admin bool
}

// TODO: Get rid of this
// Get whether the user is an admin.
// The list of admins is located at data/admins.txt.
func (u UserData) Admin() bool {
	// Is the email in admins list?
	if adminsData, err := ioutil.ReadFile(AdminListPath); err == nil {
		for _, line := range strings.Split(string(adminsData), "\n") {
			if u.Email == line {
				return true
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error reading admins list: %s\n", err.Error())
	}
	return false
}

/*
// Returns the total number of hours that a user logged
func (u UserData) Total() uint {
	return DBTotal(u.Email, u.Grade)
}

// Gets the entries of a user
func (u UserData) Entries() []*DBEntry {
	return DBList(u.Email, u.Grade)
}

// Returns whether the user can edit an entry
func (u UserData) CanEdit(entry *DBEntry) bool {
	return u.Admin() || entry.Editable()
}
*/
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

// TODO: THis should _probably_ go in Database
// Creates a user from an email
func UserFromEmail(email string) UserData {
	data, ok := StudentList[email]
	if ok {
		return data
	} else {
		return UserData{Email: email, Name: email, Grade: 0}
	}
}

// students.csv code below
// students.csv has the format Name, Grade / Graduation Year, Email

// List of students taken from data/students.csv
var StudentList = make(map[string]UserData)

// Initialize student list. Reads from data/students.csv
func StudentListInit() error {
	file, err := os.Open(StudentListPath)
	if err != nil {
		return err
	}
	reader := csv.NewReader(file)
	reader.Read() // skip header
	for {
		record, err := reader.Read()

		if err == io.EOF {
			break
		}

		if len(record) < 4 {
			fmt.Fprintln(os.Stderr, "warning: not enough cells in row '"+strings.Join(record, ",")+"'")
			continue
		}

		if err != nil && err != csv.ErrFieldCount {
			return err
		}

		grade, err := strconv.ParseUint(record[1], 10, 64)
		if err != nil {
			return err
		}

		late, err := strconv.ParseUint(record[3], 10, 8)
		if err != nil {
			return err
		}

		StudentList[record[2]] = UserData{
			Name:  record[0],
			Grade: uint(grade),
			Email: record[2],
			Late:  uint(late),
		}
	}

	return nil
}
