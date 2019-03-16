package main

import (
	"strconv"
	"errors"
	"net/http"
	"math/rand"
)

// Maps session IDs to users
var SigninMap = make(map[string]UserData)

// Generates a session ID
func SigninGen() string {
	out := strconv.FormatInt(rand.Int63(), 36)
	if _, ok := SigninMap[out]; ok { // if seesion id already is a thing
		return SigninGen() // try again
	}
	return out
}

// Signs in a user and returns the session ID
func Signin(user UserData) string {
	sid := SigninGen()
	SigninMap[sid] = user
	return sid
}

// Signs out a user
func Signout(sid string) {
	delete(SigninMap, sid)
}

// Returns user associated with sid
func SignedUser(sid string) (UserData, error) {
	user, ok := SigninMap[sid]
	if ok {
		return user, nil
	} else {
		return UserData{}, errors.New("Invalid session ID")
	}
}

// Returns the SID of a request
func SignedUserHTTP(r *http.Request) (UserData, error) {
	sidc, err := r.Cookie("BBCS_SESSION_ID")
	if err != nil {
		return UserData{}, err
	}
	return SignedUser(sidc.Value)
}
