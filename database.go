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
	return path.Base(ref.Path), err
}

// Method Set updates an entry.
func (dab *Database) Set(email string, key string, entry *Entry) error {
	ref := dab.db.NewRef("/entries").Child(dbCodeEmail(email)).Child(key)
	return ref.Set(dab.ctx, entry)
}

// Method Flag flags or unflags an entry.
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

func (dab *Database) ListAll() (map[string]EntryList, error) {
	out := make(map[string]EntryList)
	query := dab.db.NewRef("/entries").OrderByKey()
	err := query.Get(dab.ctx, &out)
	if err != nil {
		return nil, err
	}

	for codedEmail, list := range out {
		out[dbDecodeEmail(codedEmail)] = list
		delete(out, codedEmail)
	}

	return out, nil
}

// Method User returns a user.
func (dab *Database) User(email string) User {
	user := User{Name: email}
	dab.db.NewRef("/users").Child(dbCodeEmail(email)).Get(dab.ctx, &user)
	user.Email = email
	return user
}

func (dab *Database) Users() (map[string]User, error) {
	m := make(map[string]User)
	query := dab.db.NewRef("/users").OrderByKey()
	err := query.Get(dab.ctx, &m)

	for email, user := range m {
		delete(m, email)
		user.Email = dbDecodeEmail(email)
		m[user.Email] = user
	}

	return m, err
}

// Deletes all non-Admin users and adds all users specified in here.
func (dab *Database) SetStudents(users []User) error {
	usersRef := dab.db.NewRef("/users")
	return usersRef.Transaction(dab.ctx, db.UpdateFn(func(node db.TransactionNode) (interface{}, error) {
		oldUsers := make(map[string]User, 0)
		err := node.Unmarshal(&oldUsers)
		if err != nil {
			return nil, err
		}

		for codedEmail, oldUser := range oldUsers {
			if !oldUser.Admin {
				delete(oldUsers, codedEmail)
			}
		}

		for _, user := range users {
			if len(user.Email) == 0 {
				continue
			}

			oldUser := oldUsers[dbCodeEmail(user.Email)]
			user.Admin = oldUser.Admin
			oldUsers[dbCodeEmail(user.Email)] = user
		}

		return oldUsers, nil
	}))
}

func (dab *Database) Flagged() (map[[2]string]*Entry, error) {
	entries := make(map[string]EntryList)
	err := dab.db.NewRef("/entries").OrderByKey().Get(dab.ctx, &entries)
	if err != nil {
		return nil, err
	}

	m := make(map[[2]string]*Entry)
	for email, list := range entries {
		for key, entry := range list {
			if entry.Flagged {
				m[[2]string{dbDecodeEmail(email), key}] = entry
			}
		}
	}

	return m, nil
}
