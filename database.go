package main

import (
	"time"
)

type DBEntry struct {
	Name string
	Hours uint
	Date time.Time
	Organization string
	ContactName string
	ContactEmail string
}

var _dbEntries = make(map[string][]DBEntry)

/* Adds */
func DBSet(email string, entry DBEntry, index int) {
	if index < 0 {
		_dbEntries[email] = append(_dbEntries[email], entry)
	} else if index < len(_dbEntries[email]) {
		_dbEntries[email][index] = entry
	}
}

func DBList(email string) []DBEntry {
	return _dbEntries[email]
}

func DBGet(email string, index int) DBEntry {
	if len(_dbEntries[email]) > index && index >= 0 {
		return _dbEntries[email][index]
	}

	return DBEntry{
		Date: time.Now(),
		Hours: 1,
	}
}
