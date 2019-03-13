package main

import (
	"net/http"
	"errors"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"strings"
	"os"
)

type UserData struct {
	Name string
	Email string
	Admin bool
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

	// Create UserData struct; fmt.Sprint converts things to strings (just in case it's not a string)
	out := UserData{Admin:false}
	out.Email = fmt.Sprint(data["email"])
	if out.Email != "bstarr@blindbrook.org" {
		out.Name = fmt.Sprint(data["name"])
	} else {
		out.Name = "Robert Starr"
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
	return UserData{Email:email, Name:email, Admin:false}
}

/* Returns a new user created from email if u.Admin, otherwise returns u */
func (u UserData) IfAdmin(email string) UserData {
	if u.Admin && len(email) != 0 && email != u.Email {
		return UserFromEmail(email)
	} else {
		return u
	}
}