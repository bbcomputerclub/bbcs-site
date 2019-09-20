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
	Admin bool
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

/*
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
*/
