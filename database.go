package main

/* Database functions
 *
 * Contains the functions necessary for interfacing with the database
 */

import (
	"context"
	"errors"
	"firebase.google.com/go"
	"firebase.google.com/go/db"
	"fmt"
	"google.golang.org/api/option"
	"path"
	"sort"
	"strings"
)

var EntryNotFound = errors.New("entry not found")

func dbCodeEmail(email string) string {
	return strings.Replace(email, ".", "^", -1)
}

func dbDecodeEmail(code string) string {
	return strings.Replace(code, "^", ".", -1)
}

// Type Database might be thread-safe. We don't know.
type Database struct {
	app *firebase.App
	db  *db.Client
	ctx context.Context
}

// Function NewDatabase creates a new Database.
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

// Method Get returns the entry if it does exist and and error otherwise
func (dab *Database) Get(email string, key string) (*Entry, error) {
	entry := new(Entry)
	err := dab.db.NewRef("/entries").Child(dbCodeEmail(email)).Child(key).Get(dab.ctx, entry)
	if err != nil {
		return nil, err
	}
	if entry.Name == "" {
		return nil, EntryNotFound
	}
	return entry, nil
}

// Method Add should be self-explanatory.
func (dab *Database) Add(email string, entry *Entry) (string, error) {
	ref, err := dab.db.NewRef("/entries").Child(dbCodeEmail(email)).Push(dab.ctx, entry)
	if err != nil {
		return "", err
	}
	return path.Base(ref.Path), nil
}

// Method Set updates an entry.
func (dab *Database) Set(email string, key string, entry *Entry) error {
	ref := dab.db.NewRef("/entries").Child(dbCodeEmail(email)).Child(key)
	return ref.Set(dab.ctx, entry)
}

// Method Flag flags or unfalgs an entry.
func (dab *Database) Flag(email string, key string, flag bool) error {
	ref := dab.db.NewRef("/entries").Child(dbCodeEmail(email)).Child(key)
	return ref.Update(dab.ctx, map[string]interface{}{
		"flagged": flag,
	})
}

// Method Remove removes an entry.
func (dab *Database) Remove(email string, key string) error {
	ref := dab.db.NewRef("/entries").Child(dbCodeEmail(email)).Child(key)
	return ref.Delete(dab.ctx)
}

// Method List returns a list of a person's entries.
func (dab *Database) List(email string) (EntryList, error) {
	list := make(EntryList)
	query := dab.db.NewRef("/entries").Child(dbCodeEmail(email)).OrderByKey()
	err := query.Get(dab.ctx, &list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// Method ListSorted returns a list and a slice containing the keys in the correct order.
func (dab *Database) ListSorted(email string) ([]string, EntryList, error) {
	list, err := dab.List(email)
	if err != nil {
		return nil, nil, err
	}
	keylist := []string(nil)
	for key, _ := range list {
		keylist = append(keylist, key)
	}

	sort.Slice(keylist, func(i, j int) bool {
		return list[keylist[i]].Date.After(list[keylist[j]].Date)
	})

	return keylist, list, nil
}

// Method User returns a user.
func (dab *Database) User(email string) UserData {
	user := UserData{Email: email, Name: email}
	dab.db.NewRef("/students").Child(dbCodeEmail(email)).Get(dab.ctx, &user)
	return user
}
