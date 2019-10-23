package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"sync"
)

// Type TokenMap represents a token map.
type TokenMap struct {
	m     map[string]User
	mutex *sync.RWMutex
}

func NewTokenMap() *TokenMap {
	return &TokenMap{
		m:     make(map[string]User),
		mutex: new(sync.RWMutex),
	}
}

// Doesn't lock
func (m *TokenMap) newToken() string {
	num, err := rand.Int(rand.Reader, new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil))
	if err != nil {
		return m.newToken()
	}
	token := num.Text(36)
	if _, ok := m.m[token]; ok {
		return m.newToken()
	}
	return token
}

func (m *TokenMap) AddGToken(gtoken string, database *Database) (string, User, error) {
	user, err := tmUserFromGToken(gtoken, database)
	if err != nil {
		return "", User{}, err
	}
	return m.Add(user), user, nil
}

func (m *TokenMap) Add(user User) string {
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

func (m *TokenMap) Get(token string) (User, bool) {
	m.mutex.RLock()
	user, ok := m.m[token]
	m.mutex.RUnlock()
	return user, ok
}

// Takes in a Google Token and returns a User.
func tmUserFromGToken(token string, database *Database) (User, error) {
	// Next 8 lines: Retrieves data from Google servers
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + token)
	if err != nil {
		return User{}, errors.New("something went wrong")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return User{}, errors.New("something went wrong")
	}

	// The data is stored in JSON. Unmarshal the data
	data := make(map[string]interface{})
	json.Unmarshal(body, &data)

	// If the error field is present, there is an error
	if data["error"] != nil {
		return User{}, errors.New("not signed in: " + fmt.Sprint(data["error"]))
	}

	// Make sure the domain is Blind Brook (the account is from Blind Brook)
	if fmt.Sprint(data["hd"]) != "blindbrook.org" {
		return User{}, errors.New("that account isn't associated with Blind Brook")
	}
	out := database.User(fmt.Sprint(data["email"]))

	// Create User struct if not found in map; fmt.Sprint converts things to strings (just in case it's not a string)
	if out.Name == out.Email || out.Name == "" {
		out.Email = fmt.Sprint(data["email"])
		out.Name = fmt.Sprint(data["name"])
	}

	// TODO: Check if it's from this app

	return out, nil
}
