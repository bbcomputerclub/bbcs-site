package main

/* Database functions
 *
 * Contains the functions necessary for interfacing with the database
 */

import (
	"context"
	"encoding/json"
	"firebase.google.com/go"
	"firebase.google.com/go/db"
	"fmt"
	"google.golang.org/api/option"
	"io/ioutil"
	"net/url"
	"os"
	"path"
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

func dbCodeEmail(email string) string {
	return strings.Replace(email, ".", "^", -1)
}

func dbDecodeEmail(code string) string {
	return strings.Replace(code, "^", ".", -1)
}

// Type Database maybe threadsafe?
type Database struct {
	app *firebase.App
	db  *db.Client
	ctx context.Context
}

func NewDatabase(configFile string, databaseURL string) (*Database, error) {
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(configFile))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to app: %v", err)
	}
	database, err := app.DatabaseWithURL(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %v", err)
	}
	return &Database{
		app: app,
		db:  database,
		ctx: ctx,
	}, nil
}

func (dab *Database) Get(email string, key string) (*Entry, error) {
	entry := new(Entry)
	err := dab.db.NewRef("/entries").Child(dbCodeEmail(email)).Child(key).Get(dab.ctx, &entry)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (dab *Database) Add(email string, entry *Entry) (string, error) {
	ref, err := dab.db.NewRef("/entries").Child(dbCodeEmail(email)).Push(dab.ctx, entry)
	if err != nil {
		return "", err
	}
	return path.Base(ref.Path), nil
}

func (dab *Database) Set(email string, key string, entry *Entry) error {
	ref := dab.db.NewRef("/entries").Child(dbCodeEmail(email)).Child(key)
	return ref.Set(dab.ctx, entry)
}

func (dab *Database) Flag(email string, key string, flag bool) error {
	ref := dab.db.NewRef("/entries").Child(dbCodeEmail(email)).Child(key)
	return ref.Update(dab.ctx, map[string]interface{}{
		"flagged": flag,
	})
}

func (dab *Database) Remove(email string, key string) error {
	ref := dab.db.NewRef("/entries").Child(dbCodeEmail(email)).Child(key)
	return ref.Delete(dab.ctx)
}

type EntryList []*Entry

func (l EntryList) Total() uint {
	total := uint(0)
	for _, entry := range l {
		total += entry.Hours
	}
	return total
}

func (dab *Database) List(email string) (EntryList, error) {
	slice := make(EntryList, 0)
	query := dab.db.NewRef("/entries").Child(dbCodeEmail(email)).OrderByKey()
	err := query.Get(dab.ctx, &slice)
	if err != nil {
		return nil, err
	}
	return slice, nil
}

// Returns the path of the entries for grade `grade`
func DBPath(grade uint) string {
	return "./data/entries-" + strconv.FormatUint(uint64(grade), 10) + ".json"
}

// Represents an entry
type DBEntry struct {
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

// Returns whether the entry is at least 30 days old
func (entry *DBEntry) Editable() bool {
	duration, _ := time.ParseDuration("1h")
	return time.Since(entry.Date) <= duration*24*30
}

// Set Flagged to true or false
func (entry *DBEntry) CalcFlagged() {
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

// Represents a document that contains all entries
type DBDocument map[string][]*DBEntry

/* Returns data.json or an empty document if data.jsond doesn't exist */
func DBDocumentGet(grade uint) DBDocument {
	body, err := ioutil.ReadFile(DBPath(grade))

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

func DBDocumentWrite(grade uint, doc DBDocument) error {
	newbody, err := json.Marshal(doc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	err = ioutil.WriteFile(DBPath(grade), newbody, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return err
}

/* Adds / changes an entry */
func DBSet(email string, grade uint, entry *DBEntry, index int) {
	doc := DBDocumentGet(grade)

	if index < 0 {
		doc[email] = append(doc[email], entry)
	} else if index < len(doc[email]) {
		doc[email][index] = entry
	}

	DBDocumentWrite(grade, doc)
}

/* Lists the entries */
func DBList(email string, grade uint) []*DBEntry {
	doc := DBDocumentGet(grade)
	if len(doc[email]) != 0 {
		return doc[email]
	}

	// If that does not exist, check entries-0.json (see #38)
	defaultdoc := DBDocumentGet(0)
	if len(defaultdoc[email]) != 0 {
		doc[email] = defaultdoc[email]
		delete(defaultdoc, email)
		DBDocumentWrite(grade, doc)
		DBDocumentWrite(0, defaultdoc)
		return doc[email]
	}

	return nil
}

/* The default entry. */
func DBEntryDefault() *DBEntry {
	return &DBEntry{
		Date:  time.Now(),
		Hours: 1,
	}
}

/* Retrieves an entry. If not found, returns the default entry */
func DBGet(email string, grade uint, index int) *DBEntry {
	doc := DBDocumentGet(grade)

	if len(doc[email]) > index && index >= 0 {
		return doc[email][index]
	}

	return DBEntryDefault()
}

/* Returns the total # of hours */
func DBTotal(email string, grade uint) uint {
	list := DBList(email, grade)

	total := uint(0)
	for _, entry := range list {
		if entry != nil {
			total += entry.Hours
		}
	}
	return total
}

/* Removes an entry */
func DBRemove(email string, grade uint, index int) {
	DBSet(email, grade, nil, index)
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

// Creates an entry from a url.Values
func DBEntryFromQuery(query url.Values) *DBEntry {
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

	return &DBEntry{
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
