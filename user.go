package main

import (
	"net/http"
	"errors"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"strings"
	"os"
	"io"
	"encoding/csv"
	"strconv"
)

type UserData struct {
	Name string
	Grade uint
	Email string
	Admin bool
}

func (u UserData) Total() uint {
	return DBTotal(u.Email, u.Grade)
}

func (u UserData) CanEdit(entry *DBEntry) bool {
	return u.Admin || entry.Editable()
}

/* Passes token through Google servers to validate it */
func UserFromToken(token string) (UserData, error) {
	// Next 8 lines: Retrieves data from Google servers
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + token)	
	if err != nil {
		return UserData{}, errors.New("Something went wrong. Try again.")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return UserData{}, errors.New("Something went wrong. Try again.")
	}

	// The data is stored in JSON. Unmarshal the data
	data := make(map[string]interface{})
	json.Unmarshal(body, &data)

	// If the error field is present, there is an error
	if data["error"] != nil {
		return UserData{}, errors.New("Not signed in: " + fmt.Sprint(data["error"]))
	}

	// Make sure the domain is Blind Brook (the account is from Blind Brook)
	if fmt.Sprint(data["hd"]) != "blindbrook.org" {
		return UserData{}, errors.New("That account isn't associated with Blind Brook.")
	}

	
	out, ok := StudentList[fmt.Sprint(data["email"])]
	out.Admin = false

	// Create UserData struct if not found in map; fmt.Sprint converts things to strings (just in case it's not a string)
	if !ok {
		out.Email = fmt.Sprint(data["email"])
		out.Name = fmt.Sprint(data["name"])
	}

	// Is the email in admins.txt?
	if adminsData, err := ioutil.ReadFile("admins.txt"); err == nil {
		for _, line := range strings.Split(string(adminsData), "\n") {
			if out.Email == line {
				out.Admin = true
			}			
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error reading admins.txt: %s\n", err.Error())
	}

	return out, nil
}

func UserFromEmail (email string) UserData {
	data, ok := StudentList[email]
	if ok {
		return data
	} else {
		return UserData{Email:email, Name:email, Admin:false, Grade:0}
	}
}

// students.csv code below
// students.csv has the format Name, Grade / Graduation Year, Email

var StudentList = make(map[string]UserData)

func StudentListInit() error {
	file, err := os.Open("students.csv")
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

		if len(record) < 3 {
			fmt.Fprintln(os.Stderr, "warning: not enough cells in row '" + strings.Join(record, ",") + "'")
			continue
		}

		if err != nil && err != csv.ErrFieldCount {
			return err
		}

		grade, err := strconv.ParseUint(record[1], 10, 64)
		if err != nil {
			return err	
		}

		StudentList[record[2]] = UserData{
			Name: record[0],
			Grade: uint(grade),
			Email: record[2],
			Admin: false,
		}
	}
	
	return nil
}
