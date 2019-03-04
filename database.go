package main

import (
	"time"
	"os"
	"io/ioutil"
	"encoding/json"
	"fmt"
)

const DBPath = "./data.json"

type DBEntry struct {
	Name string
	Hours uint
	Date time.Time
	Organization string
	ContactName string
	ContactEmail string
	ContactPhone uint	
}

func (entry *DBEntry) UnmarshalJSON(data []byte) error {
	m := make(map[string]interface{})
	json.Unmarshal(data, &m)
	for key, val := range m {
		switch key {
		case "name":
			entry.Name = fmt.Sprint(val)
		case "hours":
			h, ok := val.(float64)
			if ok {
				entry.Hours = uint(h)
			}
		case "date":
			entry.Date, _ = time.Parse("2006-01-02", fmt.Sprint(val))
		case "org":
			entry.Organization = fmt.Sprint(val)
		case "contact_name":
			entry.ContactName = fmt.Sprint(val)
		case "contact_email":	
			entry.ContactEmail = fmt.Sprint(val)
		case "contact_phone": 
			h, ok := val.(float64)
			if ok {
				entry.ContactPhone = uint(h)
			}
		}
	}
	return nil
}

func (entry *DBEntry) MarshalJSON() ([]byte, error) {
	out := map[string]interface{} {
		"name": entry.Name,
		"hours": entry.Hours,
		"date": entry.Date.Format("2006-01-02"),
		"org": entry.Organization,
		"contact_name": entry.ContactName,
		"contact_email": entry.ContactEmail,
	}
	if entry.ContactPhone != 0 {
		out["contact_phone"] = entry.ContactPhone	
	}
	return json.Marshal(out)
}

func (entry *DBEntry) Editable() bool {
	duration, _ := time.ParseDuration("1h")
	return time.Since(entry.Date) <= duration * 24 * 30
}

type DBDocument map[string][]*DBEntry

/* Returns data.json or an empty document if data.jsond doesn't exist */
func DBDocumentGet() DBDocument {
	body, err := ioutil.ReadFile(DBPath)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)	
		return make(DBDocument)
	}

	doc := make(DBDocument)
	err = json.Unmarshal(body, &doc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)	
	}
	return doc
}

/* Adds */
func DBSet(email string, entry *DBEntry, index int) {
	doc := DBDocumentGet()
	
	if index < 0 {
		doc[email] = append(doc[email], entry)
	} else if index < len(doc[email]) {
		doc[email][index] = entry
	}

	newbody, err := json.Marshal(doc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	err = ioutil.WriteFile(DBPath, newbody, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}

func DBList(email string) []*DBEntry {
	doc := DBDocumentGet()
	return doc[email]
}

func DBEntryDefault() *DBEntry {
	return &DBEntry{
		Date: time.Now(),
		Hours: 1,
	}
}

func DBGet(email string, index int) *DBEntry {
	doc := DBDocumentGet()

	if len(doc[email]) > index && index >= 0 {
		return doc[email][index]
	}

	return DBEntryDefault()
}

func DBTotal(email string) uint {
	list := DBList(email)

	total := uint(0)
	for _, entry := range list {
		if entry != nil {
			total += entry.Hours
		}
	}
	return total
}

func DBRemove(email string, index int) {
	DBSet(email, nil, index)
}