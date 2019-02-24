package main

import (
	"time"
)

type DBEntry struct {
	Name string
	Hours uint
	Date time.Time
	Organization uint
	ContactName string
	ContactEmail string
}

var _dbEntries = make(map[string][]DBEntry)

/* Adds */
func DBAdd(email string, entry DBEntry) {
	_dbEntries[email] = append(_dbEntries[email], entry)
}

func DBList(email string) []DBEntry {
	return _dbEntries[email]
}

func DBGet(email string, index uint) *DBEntry {
	if uint(len(_dbEntries[email])) > index {
		return &_dbEntries[email][index]
	}

	return nil
}
