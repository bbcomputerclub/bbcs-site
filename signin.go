package main

import (
	"math/rand"
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
