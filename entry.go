package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Entry struct {
	Name         string
	Hours        uint
	Date         time.Time
	Organization string
	ContactName  string
	ContactEmail string
	ContactPhone uint
	Description  string
	LastModified time.Time
	Flagged      bool
}

func NewEntry(name string, hours uint, org string) *Entry {
	return &Entry{
		Name:         name,
		Hours:        hours,
		Organization: org,
		Date:         time.Now(),
		LastModified: time.Now(),
	}
}

func EmptyEntry() *Entry {
	return &Entry{
		Hours:        1,
		Date:         time.Now(),
		LastModified: time.Now(),
	}
}

func (entry *Entry) SetFlagged() {
	entry.Flagged = false
	if entry.Hours >= 10 {
		entry.Flagged = true
		return
	}
	keywords := []string{"cit", "counselor", "camp"}
	for _, keyword := range keywords {
		if strings.Contains(strings.ToLower(entry.Name), keyword) ||
			strings.Contains(strings.ToLower(entry.Description), keyword) ||
			strings.Contains(strings.ToLower(entry.Organization), keyword) {
			entry.Flagged = true
			return
		}
	}
}

// Returns whether the entry is at least 30 days old
func (entry *Entry) Editable() bool {
	duration, _ := time.ParseDuration("1h")
	return time.Since(entry.Date) <= duration*24*30
}

func (entry *Entry) MarshalJSON() ([]byte, error) {
	out := map[string]interface{}{
		"name":          entry.Name,
		"hours":         entry.Hours,
		"date":          entry.Date.Format("2006-01-02"),
		"org":           entry.Organization,
		"last_modified": entry.LastModified.Format("2006-01-02"),
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
	if entry.Description != "" {
		out["description"] = entry.Description
	}
	if entry.Flagged {
		out["flagged"] = true
	}
	return json.Marshal(out)
}

func (entry *Entry) UnmarshalJSON(data []byte) error {
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
		case "description":
			entry.Description = fmt.Sprint(val)
		case "last_modified":
			entry.LastModified, _ = time.Parse("2006-01-02", fmt.Sprint(val))
		case "flagged":
			if valb, ok := val.(bool); ok {
				entry.Flagged = valb
			}
		}
	}
	return nil
}

func EntryFromQuery(query url.Values) *Entry {
	hours, err := strconv.ParseUint(query.Get("hours"), 10, 64)
	if err != nil {
		hours = 1
	}
	date, err := time.Parse("2006-01-02", query.Get("date"))
	if err != nil {
		date = time.Now()
	}

	contactPhone, err := strconv.ParseUint(strings.NewReplacer("-", "", "+", "", " ", "").Replace(query.Get("contactphone")), 10, 64)
	if err != nil {
		contactPhone = 0
	}

	return &Entry{
		Name:         query.Get("name"),
		Hours:        uint(hours),
		Date:         date,
		Organization: query.Get("org"),
		ContactName:  query.Get("contactname"),
		ContactEmail: query.Get("contactemail"),
		Description:  query.Get("description"),
		ContactPhone: uint(contactPhone),
		LastModified: time.Now(),
	}
}

func (entry *Entry) EncodeQuery() url.Values {
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

type EntryList map[string]*Entry

func (l EntryList) Total() uint {
	total := uint(0)
	for _, entry := range l {
		total += entry.Hours
	}
	return total
}
