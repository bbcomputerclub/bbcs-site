package main

/* Database functions
 *
 * Contains the functions necessary for interfacing with the database
 */ 

import (
	"time"
	"os"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"strings"
	"net/url"
	"strconv"
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
	}
	if entry.ContactName != "" {
		out["contact_name"] = entry.ContactName
	}
	if entry.ContactEmail != "" {
		out["contact_email"] = entry.ContactEmail
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

// Represents a document that contains all entries
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

/* Adds / changes an entry */
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

/* Lists the entries */
func DBList(email string) []*DBEntry {
	doc := DBDocumentGet()
	return doc[email]
}

/* The default entry. */
func DBEntryDefault() *DBEntry {
	return &DBEntry{
		Date: time.Now(),
		Hours: 1,
	}
}

/* Retrieves an entry. If not found, returns the default entry */
func DBGet(email string, index int) *DBEntry {
	doc := DBDocumentGet()

	if len(doc[email]) > index && index >= 0 {
		return doc[email][index]
	}

	return DBEntryDefault()
}

/* Returns the total # of hours */
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

/* Removes an entry */
func DBRemove(email string, index int) {
	DBSet(email, nil, index)
}

func (entry *DBEntry) EncodeQuery() url.Values {
	out := url.Values{}
	out.Set("name", entry.Name)
	out.Set("hours", strconv.FormatUint(uint64(entry.Hours), 10))
	out.Set("date", entry.Date.Format("2006-01-02"))
	out.Set("org", entry.Organization)
	if entry.ContactName != "" {
		out.Set("contactname", entry.ContactName)
	}
	if entry.ContactEmail != "" {
		out.Set("contactemail", entry.ContactEmail)
	}
	if entry.ContactPhone != 0 {
		out.Set("contactphone", strconv.FormatUint(uint64(entry.ContactPhone), 10))
	}
	return out
}

func DBEntryFromQuery(query url.Values) *DBEntry {
	hours, err := strconv.ParseUint(query.Get("hours"), 10, 64)
	if err != nil {
		hours = 1
	}
	date, err := time.Parse("2006-01-02", query.Get("date"))
	if err != nil {
		date = time.Now()
	}

	contactPhone, err := strconv.ParseUint(strings.NewReplacer("-", "" , "+", "", " ", "").Replace(query.Get("contactphone")), 10, 64)
	if err != nil {
		contactPhone = 0
	}

	return &DBEntry{
		Name: query.Get("name"),
		Hours: uint(hours),
		Date: date,
		Organization: query.Get("org"),
		ContactName: query.Get("contactname"),
		ContactEmail: query.Get("contactemail"),
		ContactPhone: uint(contactPhone),
	}
}
