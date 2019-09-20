// TODO: 1) change indexes to keys (int => string)
//       2) replace DB* with Database
//
//       3) create some type State struct { database, signinmap	 }

package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	DATABASE_URL       = os.Getenv("DATABASE_URL")
	DATABASE_AUTH_FILE = "credentials.json" // $DATABASE_CREDENTIALS
)

// struct State
//
// func (State) Users(request)

var (
	database *Database = nil
	tokenMap *TokenMap = NewTokenMap()
)

const (
	ACTION_VIEW = "View"
	ACTION_ADD  = "Add"
	ACTION_EDIT = "Edit"
)

var funcMap = template.FuncMap{
	"time": func(from int) time.Time {
		return time.Now().AddDate(0, 0, from)
	},
	"fmtordinal": func(in uint) string {
		if in%10 == 1 && in != 11 {
			return fmt.Sprint(in) + "st"
		}
		if in%10 == 2 && in != 12 {
			return fmt.Sprint(in) + "nd"
		}
		if in%10 == 3 && in != 13 {
			return fmt.Sprint(in) + "rd"
		}
		return fmt.Sprint(in) + "th"
	},
}

func init() {
	credentials := os.Getenv("DATABASE_CREDENTIALS")
	file, err := os.Create(DATABASE_AUTH_FILE)
	if err != nil {
		panic(err)
	}
	_, err = io.WriteString(file, credentials)
	if err != nil {
		panic(err)
	}
	file.Close()

	database, err = NewDatabase(DATABASE_AUTH_FILE, DATABASE_URL)
	if err != nil {
		panic(err)
	}
}

func getToken(r *http.Request) string {
	if cookie, err := r.Cookie("BBCS_SESSION_ID"); err == nil {
		return cookie.Value
	}
	return ""
}

func main() {
	rand.Seed(time.Now().UnixNano())

	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "files/login.html")
	})

	r.HandleFunc("/icons/{file:.[0-9]+\\.(?:svg|png)}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		filename := vars["file"]

		if strings.HasSuffix(filename, ".png") {
			/* PNG
			 * PNG files are stored in the directory as icon-N.png
			 * This section simply retrieves the file and serves it
			 */
			file, err := os.Open("icons/icon-" + filename)
			if err != nil {
				w.WriteHeader(404)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			io.Copy(w, file)
			return
		} else if strings.HasSuffix(vars["file"], ".svg") {
			bodybyte, err := ioutil.ReadFile("icons/icon.svg")
			if err != nil {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(500)
				return
			}
			body := string(bodybyte)
			size := filename[0 : len(filename)-4]

			// Replace width and height attributes
			i := strings.Index(body, "width=\"")
			j := strings.Index(body[i+7:], "\"") + i + 7
			body = body[0:i+7] + size + body[j:]

			i = strings.Index(body, "height=\"")
			j = strings.Index(body[i+8:], "\"") + i + 8
			body = body[0:i+8] + size + body[j:]

			// Serve
			w.Header().Set("Content-Type", "image/svg+xml")
			w.Write([]byte(body))
			return
		}
		w.WriteHeader(404)
	})

	r.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "files/style.css")
	})

	r.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "files/manifest.json")
	})

	r.HandleFunc("/generator", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "files/generator.html")
	})

	r.Handle("/source", http.RedirectHandler("https://github.com/bbcomputerclub/bbcs-site", 301))

	/* GET /signin
	 *
	 * Signs the user in.
	 * Parameters:
	 * 	  token: The token
	 *    redirect: An escaped URI
	 */
	// TODO: This should be POST
	r.HandleFunc("/signin", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		token, user, err := tokenMap.AddGToken(query.Get("token"))
		if err != nil {
			w.Header().Set("Refresh", "0; url=/#error:"+url.QueryEscape(err.Error()))
			w.WriteHeader(403)
			return
		}

		http.SetCookie(w, &http.Cookie{Name: "BBCS_SESSION_ID", Path: "/", Value: token, HttpOnly: true})

		redirect, err := url.QueryUnescape(query.Get("redirect"))
		if len(redirect) == 0 || err != nil {
			if user.Admin() {
				w.Header().Set("Location", "/all")
			} else {
				w.Header().Set("Location", "/"+user.Email)
			}
		} else {
			w.Header().Set("Location", redirect)
		}
		w.WriteHeader(303)
	})

	// TODO: This should be POST
	r.HandleFunc("/signout", func(w http.ResponseWriter, r *http.Request) {
		token := getToken(r)
		tokenMap.Remove(token)

		http.SetCookie(w, &http.Cookie{Name: "BBCS_SESSION_ID", Value: "", MaxAge: -1})
		w.Header().Set("Location", "/#signout")
		w.WriteHeader(303)
	})

	// Updates an entry
	r.HandleFunc("/do/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" && r.Method != "PUT" {
			w.Header().Set("Allow", "POST, PUT")
			io.WriteString(w, "invalid request")
			w.WriteHeader(405)
			return
		}

		user, ok := tokenMap.Get(getToken(r))
		if !ok {
			w.WriteHeader(403)
			io.WriteString(w, "invalid token")
			return
		}
		email := r.PostFormValue("user")
		if !user.Admin() && user.Email != email {
			w.WriteHeader(403)
			io.WriteString(w, "not enough permissions")
			return
		}

		// Get entry
		key := r.PostFormValue("entry")
		oldEntry, _ := database.Get(email, key)
		newEntry := EntryFromQuery(r.PostForm)

		// Make sure entry is editable
		if (!user.Admin()) && (!oldEntry.Editable() || !newEntry.Editable()) {
			w.WriteHeader(403)
			io.WriteString(w, "not enough permissions")
			return
		}
		newEntry.SetFlagged()

		// Make changes
		database.Set(email, key, newEntry)

		// Redirect
		w.Header().Set("Location", "/"+email)
		w.WriteHeader(302)
	})

	// Adds an entry
	r.HandleFunc("/do/add", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" && r.Method != "PUT" {
			w.Header().Set("Allow", "POST, PUT")
			io.WriteString(w, "invalid request")
			w.WriteHeader(405)
			return
		}

		user, ok := tokenMap.Get(getToken(r))
		if !ok {
			w.WriteHeader(403)
			io.WriteString(w, "invalid token")
			return
		}
		email := r.PostFormValue("user")

		if !user.Admin() && email != user.Email {
			w.WriteHeader(403)
			io.WriteString(w, "not enough permissions")
			return
		}

		// Get entry
		newEntry := EntryFromQuery(r.PostForm)

		// Make sure entry is editable
		if !user.Admin() && !newEntry.Editable() {
			w.WriteHeader(403)
			io.WriteString(w, "not enough permissions")
			return
		}
		newEntry.SetFlagged()

		// Make changes
		database.Add(email, newEntry)

		// Redirect
		w.Header().Set("Location", "/"+email)
		w.WriteHeader(302)
	})

	// Removes an entry
	r.HandleFunc("/do/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" && r.Method != "DELETE" {
			w.Header().Set("Allow", "POST, DELETE")
			w.WriteHeader(405)
			return
		}

		user, ok := tokenMap.Get(getToken(r))
		if !ok {
			w.WriteHeader(403)
			io.WriteString(w, "invalid token")
			return
		}
		email := r.PostFormValue("user")

		// Get entry
		key := r.PostFormValue("entry")
		entry, err := database.Get(email, key)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		// Make sure existing entry is editable
		if !user.Admin() && !entry.Editable() {
			w.WriteHeader(403)
			return
		}

		// Make changes
		database.Remove(email, key)

		// Redirect
		w.Header().Set("Location", "/list?user="+email)
		w.WriteHeader(302)
	})

	// Marks an entry as not suspicious
	r.HandleFunc("/do/unflag", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" && r.Method != "PU	T" {
			w.Header().Set("Allow", "POST, PUT")
			w.WriteHeader(405)
			return
		}

		user, ok := tokenMap.Get(getToken(r))
		if !ok || !user.Admin() {
			w.WriteHeader(403)
			return
		}
		student := database.User(r.PostFormValue("user"))

		// Get entry
		key := r.PostFormValue("entry")

		// Make changes
		database.Flag(student.Email, key, false)

		// Redirect
		w.Header().Set("Location", "/flagged")
		w.WriteHeader(302)
	})

	r.HandleFunc("/all", func(w http.ResponseWriter, r *http.Request) {

	})

	r.HandleFunc("/all/flagged", func(w http.ResponseWriter, r *http.Request) {

	})

	r.HandleFunc("/{email}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		email := vars["email"]

		temp, err := template.New("list.html").Funcs(funcMap).ParseFiles("files/list.html")
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}

		user, ok := tokenMap.Get(getToken(r))
		if !ok || (!user.Admin() && user.Email != email) {
			w.WriteHeader(403)
			io.WriteString(w, "invalid token or not enough permissions")
			return
		}

		keys, entries, err := database.ListSorted(email)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		data := map[string]interface{}{
			"User":    user,
			"Student": database.User(email),
			"Entries": entries,
			"Keys":    keys,
		}

		if err := temp.Execute(w, data); err != nil {
			log.Println(err)
		}
	})

	r.HandleFunc("/{email}/{key}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		email := vars["email"]
		key := vars["key"]

		temp, err := template.New("edit.html").Funcs(funcMap).ParseFiles("files/edit.html")
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}

		user, ok := tokenMap.Get(getToken(r))
		if !ok {
			w.WriteHeader(403)
			io.WriteString(w, "invalid token")
			return
		}

		var entry *Entry
		if key == "add" {
			entry = EmptyEntry()
			// TODO: EntryFromQuery
		} else {
			entry, err = database.Get(email, key)
			if err != nil {
				w.WriteHeader(404)
				io.WriteString(w, "entry not found")
				return
			}
		}

		action := ""
		switch {
		case key == "add":
			action = ACTION_ADD
		case entry.Editable() || user.Admin():
			action = ACTION_EDIT
		default:
			action = ACTION_VIEW
		}

		data := map[string]interface{}{
			"User":    user,
			"Student": database.User(email),
			"Entry":   entry,
			"Key":     key,
			"Action":  action,
		}

		if err := temp.Execute(w, data); err != nil {
			log.Println(err)
		}
	})

	r.HandleFunc("/{email}/{key}/duplicate", func(w http.ResponseWriter, r *http.Request) {
		// TODO
	})

	port := os.Getenv("PORT")
	if port == "" {
		panic("$PORT must be set")
	}

	fmt.Printf("http://localhost:%s/\n", port)

	if err := StudentListInit(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
		return
	}

	err := http.ListenAndServe(":"+port, r)
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
