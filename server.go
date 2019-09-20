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
	"path/filepath"
	"strings"
	"time"
)

var (
	DATABASE_URL       = os.Getenv("DATABASE_URL")
	DATABASE_AUTH_FILE = "credentials.json" // $DATABASE_CREDENTIALS
)

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
		if in%10 == 1 && in%100 != 11 {
			return fmt.Sprint(in) + "st"
		}
		if in%10 == 2 && in%100 != 12 {
			return fmt.Sprint(in) + "nd"
		}
		if in%10 == 3 && in%100 != 13 {
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

// Type ActionHandler represents a POST request handler.
//
// Passes the student's email as the first argument. If the user is not authenticated or
// the user does not have sufficient permissions, an empty string is passed as the first argument.
//
// The 2nd argument is the signed-in user, and the 3rd argument is the original request.
type ActionHandler func(student string, user UserData, query url.Values) (uint16, string)

func (f ActionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(405)
		return
	}

	user, ok := tokenMap.Get(getToken(r))
	if !ok {
		w.Header().Set("Refresh", "0;url=/")
		w.WriteHeader(401)
		return
	}
	email := ""
	if user.Admin || r.PostFormValue("user") == user.Email {
		email = r.PostFormValue("user")
	}

	status, loc := f(email, user, r.PostForm)
	if status >= 300 && status < 400 && loc != "" {
		w.Header().Set("Location", loc)
	}
	w.WriteHeader(int(status))
}

//
type TemplateHandler func(student string, user UserData, query url.Values, vars map[string]string) (code uint16, path string, data interface{})

func (f TemplateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(405)
	}

	user, ok := tokenMap.Get(getToken(r))
	if !ok {
		w.Header().Set("Refresh", "0;url=/")
		w.WriteHeader(401)
		return
	}

	vars := mux.Vars(r)
	email := ""
	if user.Admin || user.Email == vars["email"] {
		email = vars["email"]
	}

	code, path, data := f(email, user, r.URL.Query(), vars)

	if code < 200 || code >= 300 {
		w.WriteHeader(int(code))
		return
	}

	temp, err := template.New(filepath.Base(path)).Funcs(funcMap).ParseFiles(path)
	if err != nil {
		log.Printf("error parsing %s: %s", path, err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(int(code))
	if err = temp.Execute(w, data); err != nil {
		log.Printf("error serving %s: %s", path, err)
	}
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

		token, user, err := tokenMap.AddGToken(query.Get("token"), database)
		if err != nil {
			w.Header().Set("Refresh", "0; url=/#error:"+url.QueryEscape(err.Error()))
			w.WriteHeader(403)
			return
		}

		http.SetCookie(w, &http.Cookie{Name: "BBCS_SESSION_ID", Path: "/", Value: token, HttpOnly: true})

		redirect, err := url.QueryUnescape(query.Get("redirect"))
		if len(redirect) == 0 || err != nil {
			if user.Admin {
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

	// Redirects to /{email}/add
	r.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		user, ok := tokenMap.Get(getToken(r))
		if !ok {
			w.WriteHeader(403)
			return
		}
		w.Header().Set("Location", "/"+user.Email+"/add?"+r.URL.RawQuery)
		w.WriteHeader(303)
	})

	// Updates an entry
	r.Handle("/do/update", ActionHandler(func(email string, user UserData, query url.Values) (uint16, string) {
		if email == "" {
			return 403, ""
		}

		// Get entry
		key := query.Get("entry")
		oldEntry, _ := database.Get(email, key)
		newEntry := EntryFromQuery(query)

		// Make sure entry is recent
		if (!user.Admin) && (!oldEntry.Editable() || !newEntry.Editable()) {
			return 403, ""
		}

		newEntry.SetFlagged()

		database.Set(email, key, newEntry)

		return 303, "/" + email
	}))

	// Adds an entry
	r.Handle("/do/add", ActionHandler(func(email string, user UserData, query url.Values) (uint16, string) {
		if email == "" {
			return 403, ""
		}

		newEntry := EntryFromQuery(query)

		// Make sure entry is recent
		if !user.Admin && !newEntry.Editable() {
			return 403, ""
		}
		newEntry.SetFlagged()

		database.Add(email, newEntry)

		// Redirect
		return 303, "/" + email
	}))

	// Removes an entry
	r.Handle("/do/delete", ActionHandler(func(email string, user UserData, query url.Values) (uint16, string) {
		if email == "" {
			return 403, ""
		}

		// Get entry
		key := query.Get("entry")
		entry, err := database.Get(email, key)
		if err != nil {
			return 403, ""
		}

		// Make sure existing entry is editable
		if !user.Admin && !entry.Editable() {
			return 403, ""
		}

		// Make changes
		database.Remove(email, key)

		// Redirect
		return 303, "/" + email
	}))

	// Marks an entry as not suspicious
	r.Handle("/do/unflag", ActionHandler(func(email string, user UserData, query url.Values) (uint16, string) {
		if !user.Admin || email == "" {
			return 403, ""
		}

		key := query.Get("entry")
		database.Flag(email, key, false)
		return 303, "/all/flagged"
	}))

	r.Handle("/all", TemplateHandler(func(email string, user UserData, query url.Values, vars map[string]string) (uint16, string, interface{}) {
		if !user.Admin {
			return 403, "", nil
		}

		users, err := database.Users()
		if err != nil {
			return 500, "", nil
		}

		totals := make(map[string]uint)
		grades := make([]uint, 0, len(users))
		for grade, users := range users {
			grades = append(grades, grade)
			for _, student := range users {
				entries, err := database.List(student.Email)
				if err != nil {
					continue
				}
				totals[student.Email] = entries.Total()
			}
		}

		return 200, "files/admin.html", map[string]interface{}{
			"User":     user,
			"Students": users,
			"Grades":   grades,
			"Totals":   totals,
		}
	}))

	r.Handle("/all/flagged", TemplateHandler(func(email string, user UserData, query url.Values, vars map[string]string) (uint16, string, interface{}) {
		return 500, "", nil
	}))

	r.Handle("/{email}", TemplateHandler(func(email string, user UserData, query url.Values, vars map[string]string) (uint16, string, interface{}) {
		keys, entries, err := database.ListSorted(email)
		if err != nil {
			return 404, "", nil
		}

		return 200, "files/list.html", map[string]interface{}{
			"User":    user,
			"Student": database.User(email),
			"Entries": entries,
			"Keys":    keys,
		}
	}))

	r.Handle("/{email}/{key}", TemplateHandler(func(email string, user UserData, query url.Values, vars map[string]string) (uint16, string, interface{}) {
		if email == "" {
			return 403, "", nil
		}

		key := vars["key"]

		var entry *Entry
		if key == "add" {
			entry = EntryFromQuery(query)
		} else {
			var err error
			entry, err = database.Get(email, key)
			if err != nil {
				return 404, "", nil
			}
		}

		action := ""
		switch {
		case key == "add":
			action = ACTION_ADD
		case entry.Editable() || user.Admin:
			action = ACTION_EDIT
		default:
			action = ACTION_VIEW
		}

		return 200, "files/edit.html", map[string]interface{}{
			"User":    user,
			"Student": database.User(email),
			"Entry":   entry,
			"Key":     key,
			"Action":  action,
		}
	}))

	r.HandleFunc("/{email}/{key}/duplicate", func(w http.ResponseWriter, r *http.Request) {
		user, ok := tokenMap.Get(getToken(r))
		if !ok {
			w.WriteHeader(403)
			return
		}
		vars := mux.Vars(r)
		key := vars["key"]
		email := vars["email"]
		if !user.Admin && email != user.Email {
			w.WriteHeader(403)
			return
		}

		entry, err := database.Get(email, key)
		if err != nil {
			w.WriteHeader(404)
			return
		}
		entry.Date = time.Now()

		w.Header().Set("Location", "/add?"+entry.EncodeQuery().Encode())
		w.WriteHeader(303)
	})

	port := os.Getenv("PORT")
	if port == "" {
		panic("$PORT must be set")
	}

	fmt.Printf("http://localhost:%s/\n", port)

	err := http.ListenAndServe(":"+port, r)
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
