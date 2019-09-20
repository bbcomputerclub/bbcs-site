package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
)

type TokenMap struct {
	m     map[string]UserData
	mutex *sync.RWMutex
}

func NewTokenMap() *TokenMap {
	return &TokenMap{
		m:     make(map[string]UserData),
		mutex: new(sync.RWMutex),
	}
}

// Doesn't lock
func (m *TokenMap) newToken() string {
	token := strconv.FormatInt(rand.Int63(), 36)
	if _, ok := m.m[token]; ok {
		return m.newToken()
	}
	return token
}

func (m *TokenMap) AddGToken(gtoken string) (string, UserData, error) {
	user, err := tmUserFromGToken(gtoken)
	if err != nil {
		return "", UserData{}, err
	}
	return m.Add(user), user, nil
}

func (m *TokenMap) Add(user UserData) string {
	m.mutex.Lock()
	token := m.newToken()
	m.m[token] = user
	m.mutex.Unlock()
	return token
}

func (m *TokenMap) Remove(token string) {
	m.mutex.Lock()
	delete(m.m, token)
	m.mutex.Unlock()
}

func (m *TokenMap) Get(token string) (UserData, bool) {
	m.mutex.RLock()
	user, ok := m.m[token]
	m.mutex.RUnlock()
	return user, ok
}

func tmUserFromGToken(token string) (UserData, error) {
	// Next 8 lines: Retrieves data from Google servers
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + token)
	if err != nil {
		return UserData{}, errors.New("something went wrong")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return UserData{}, errors.New("something went wrong")
	}

	// The data is stored in JSON. Unmarshal the data
	data := make(map[string]interface{})
	json.Unmarshal(body, &data)

	// If the error field is present, there is an error
	if data["error"] != nil {
		return UserData{}, errors.New("not signed in: " + fmt.Sprint(data["error"]))
	}

	// Make sure the domain is Blind Brook (the account is from Blind Brook)
	if fmt.Sprint(data["hd"]) != "blindbrook.org" {
		return UserData{}, errors.New("that account isn't associated with Blind Brook")
	}

	out, ok := StudentList[fmt.Sprint(data["email"])]

	// Create UserData struct if not found in map; fmt.Sprint converts things to strings (just in case it's not a string)
	if !ok {
		out.Email = fmt.Sprint(data["email"])
		out.Name = fmt.Sprint(data["name"])
	}

	// TODO: Check if it's from this app

	return out, nil
}
