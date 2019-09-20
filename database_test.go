package main

import (
	"testing"
	"time"
)

func TestDatabase(t *testing.T) {
	entry := NewEntry("Soup Kitchen", 2, "Blind Brook Soup Kitchen")
	key, err := database.Add("test@example.org", entry)
	if err != nil {
		t.Error(err)
		return
	}
	err = database.Flag("test@example.org", key, true)
	if err != nil {
		t.Error(err)
		return
	}
	<-time.After(5 * time.Second)
	err = database.Remove("test@example.org", key)
	if err != nil {
		t.Error(err)
		return
	}
}
