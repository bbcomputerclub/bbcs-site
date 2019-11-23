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
	"sort"
	"strings"
	"time"
)

var (
	CLIENT_ID    = os.Getenv("BBCS_CLIENT_ID")
	DATABASE_URL = os.Getenv("DATABASE_URL")
	// DATABASE_CREDENTIALS = content of the JSON key file generated by Firebase
	DOMAIN             = os.Getenv("BBCS_DOMAIN")
	DATABASE_AUTH_FILE = "credentials.json"
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
	"dict": func(in ...interface{}) map[string]interface{} {
		m := make(map[string]interface{})
		for index, arg := range in {
			if index%2 == 1 {
				m[in[index-1].(string)] = arg
			}
		}
		return m
	},
}

func init() {
	if CLIENT_ID == "" {
		panic("$BBCS_CLIENT_ID must be set")
	}

	if DOMAIN == "" {
		panic("$BBCS_DOMAIN must be set")
	}

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

// Alias ActionHandlerFunc is used for ActionHandler.
//
// Passes the student's email as the first argument. If the user is not authenticated or
// the user does not have sufficient permissions, an empty string is passed as the first argument.
//
// The 2nd argument is the signed-in user, and the 3rd argument is the original request.
type ActionHandlerFunc = func(student string, user User, query url.Values, w http.ResponseWriter, r *http.Request) (uint16, string, error)

// Type ActionHandler represents a POST request handler.
type ActionHandler struct {
	Func         ActionHandlerFunc
	RequireAuth  bool
	RequireAdmin bool
}

func NewActionHandler(reqAuth, reqAdmin bool, f ActionHandlerFunc) ActionHandler {
	return ActionHandler{
		Func:         f,
		RequireAuth:  reqAuth,
		RequireAdmin: reqAdmin,
	}
}

func (h ActionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(405)
		return
	}

	user := User{}
	student := ""
	if h.RequireAuth {
		var ok bool
		user, ok = tokenMap.Get(getToken(r))
		if !ok {
			w.WriteHeader(401)
			return
		}

		if user.Admin || r.PostFormValue("user") == user.Email {
			student = r.PostFormValue("user")
		}
	}

	if h.RequireAdmin && !user.Admin {
		w.WriteHeader(403)
		return
	}

	status, loc, err := h.Func(student, user, r.PostForm, w, r)
	if err != nil {
		w.WriteHeader(int(status))
		io.WriteString(w, err.Error())
		return
	}
	if loc != "" {
		if status >= 300 && status < 400 {
			w.Header().Set("Location", loc)
		} else {
			w.Header().Set("Refresh", "0;url="+loc)
		}
	}
	w.WriteHeader(int(status))
}

var HEAD_TEMPLATE = template.Must(template.New("").Funcs(funcMap).ParseFiles("files/fields.html", "files/head.html", "files/toolbar.html"))

// Alias TemplateHandlerFunc represents a handler function for pages.
//
// The 1st argument is the student, 2nd is the logged-in user, 3rd is the GET query, 4th is mux.Vars.
type TemplateHandlerFunc = func(student string, user User, query url.Values, vars map[string]string) (code uint16, path string, data interface{})

// Type TemplateHandler is a Handler that is used when a template is returned.
type TemplateHandler struct {
	Func         TemplateHandlerFunc
	RequireAuth  bool
	RequireAdmin bool
}

func NewTemplateHandler(reqAuth bool, reqAdmin bool, fn TemplateHandlerFunc) TemplateHandler {
	return TemplateHandler{
		Func:         fn,
		RequireAuth:  reqAuth,
		RequireAdmin: reqAdmin,
	}
}

func (h TemplateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(405)
		return
	}

	// Authentication
	user := User{}
	if h.RequireAuth {
		var ok bool
		user, ok = tokenMap.Get(getToken(r))
		if !ok {
			w.Header().Set("Refresh", "0;url=/?"+r.URL.Path+"?"+r.URL.RawQuery)
			w.WriteHeader(401)
			return
		}
	}

	// Check for Admin-ness
	if h.RequireAdmin && !user.Admin {
		w.WriteHeader(403)
		return
	}

	vars := mux.Vars(r)
	student := ""
	if user.Admin || user.Email == vars["email"] {
		student = vars["email"]
	}

	code, path, data := h.Func(student, user, r.URL.Query(), vars)

	if code < 200 || code >= 300 {
		w.WriteHeader(int(code))
		return
	}

	temp, err := HEAD_TEMPLATE.Clone()
	if err != nil {
		log.Printf("error parsing: %s", err)
		w.WriteHeader(500)
		return
	}
	temp, err = temp.ParseFiles(path)
	if err != nil {
		log.Printf("error parsing %s: %s", path, err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(int(code))
	if err = temp.ExecuteTemplate(w, filepath.Base(path), data); err != nil {
		log.Printf("error serving %s: %s", path, err)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	r := mux.NewRouter()

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

	r.HandleFunc("/qrcode.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "files/qrcode.js")
	})

	r.Handle("/generator", NewTemplateHandler(false, false, func(email string, user User, query url.Values, vars map[string]string) (uint16, string, interface{}) {
		return 200, "files/generator.html", map[string]interface{}{
			"Entry": EntryFromQuery(query),
		}
	}))

	r.Handle("/source", http.RedirectHandler("https://github.com/bbcomputerclub/bbcs-site", 301))

	// GET /signin
	// Generates a token based on a Google token.
	r.HandleFunc("/signin", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		token, user, err := tokenMap.AddGToken(query.Get("token"), database, DOMAIN)
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
	r.Handle("/signout", NewActionHandler(false, false, func(student string, user User, query url.Values, w http.ResponseWriter, r *http.Request) (uint16, string, error) {
		token := getToken(r)
		tokenMap.Remove(token)

		http.SetCookie(w, &http.Cookie{Name: "BBCS_SESSION_ID", Value: "", MaxAge: -1})

		return 303, "/#signout", nil
	}))

	// GET /add
	// Redirects to /{email}/add
	r.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		user, ok := tokenMap.Get(getToken(r))
		if !ok {
			w.Header().Set("Refresh", "0;url=/?"+r.URL.Path+"?"+r.URL.RawQuery)
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Location", "/"+user.Email+"/add?"+r.URL.RawQuery)
		w.WriteHeader(303)
	})

	// POST /do/update
	// Updates an entry
	r.Handle("/do/update", NewActionHandler(true, false, func(email string, user User, query url.Values, _ http.ResponseWriter, _ *http.Request) (uint16, string, error) {
		if email == "" {
			return 403, "", fmt.Errorf("not logged in")
		}

		// Get entry
		key := query.Get("entry")
		oldEntry, _ := database.Get(email, key)
		newEntry := EntryFromQuery(query)

		// Make sure entry is recent
		if (!user.Admin) && (!oldEntry.Editable() || !newEntry.Editable()) {
			return 403, "", fmt.Errorf("entry too old")
		}

		newEntry.SetFlagged()

		database.Set(email, key, newEntry)

		return 303, "/" + email, nil
	}))

	// POST /do/add
	// Adds an entry
	r.Handle("/do/add", NewActionHandler(true, false, func(student string, user User, query url.Values, _ http.ResponseWriter, _ *http.Request) (uint16, string, error) {
		if student == "" {
			return 403, "", fmt.Errorf("not logged in")
		}

		newEntry := EntryFromQuery(query)

		// Make sure entry is recent
		if !user.Admin && !newEntry.Editable() {
			return 403, "", fmt.Errorf("entry too old")
		}
		newEntry.SetFlagged()

		database.Add(student, newEntry)

		// Redirect
		return 303, "/" + student, nil
	}))

	// POST /do/delete
	// Removes an entry. Only available for Admin users.
	r.Handle("/do/delete", NewActionHandler(true, true, func(student string, user User, query url.Values, _ http.ResponseWriter, _ *http.Request) (uint16, string, error) {
		if student == "" {
			return 403, "", fmt.Errorf("not logged in")
		}

		// Make changes
		key := query.Get("entry")
		database.Remove(student, key)

		// Redirect
		return 303, "/" + student, nil
	}))

	// POST /do/unflag
	// Marks an entry as not suspicious. Only available for Admin users. In fact, non-Admin users can't even view the Flagged field.
	r.Handle("/do/unflag", NewActionHandler(true, true, func(student string, user User, query url.Values, _ http.ResponseWriter, _ *http.Request) (uint16, string, error) {
		if student == "" {
			return 403, "", fmt.Errorf("no student specified")
		}

		key := query.Get("entry")
		database.Flag(student, key, false)
		return 303, "/all/flagged", nil
	}))

	// POST /do/roster
	// Updates the roster.
	r.Handle("/do/roster", NewActionHandler(true, true, func(email string, user User, query url.Values, _ http.ResponseWriter, r *http.Request) (uint16, string, error) {
		if !user.Admin {
			return 403, "", fmt.Errorf("admin permissions required")
		}

		file, _, err := r.FormFile("roster")
		if err != nil {
			log.Println(err)
			return 500, "", fmt.Errorf("internal error")
		}

		users, err := UsersFromCSV(file)
		if err != nil {
			log.Println(err)
			return 400, "", fmt.Errorf("malformed CSV file: %v", err)
		}

		err = database.SetStudents(users)
		if err != nil {
			log.Println(err)
			return 500, "", fmt.Errorf("internal error")
		}

		return 303, "/all", nil
	}))

	// GET /
	// Serves login page if user isn't logged in.
	r.Handle("/", NewTemplateHandler(false, false, func(email string, user User, query url.Values, vars map[string]string) (uint16, string, interface{}) {
		return 200, "files/login.html", map[string]interface{}{
			"ClientID": CLIENT_ID,
			"Domain":   DOMAIN,
		}
	}))

	// GET /all
	// Serves the Admin Dashboard.
	r.Handle("/all", NewTemplateHandler(true, true, func(email string, user User, query url.Values, vars map[string]string) (uint16, string, interface{}) {
		userlist, err := database.Users()
		if err != nil {
			return 500, "", nil
		}

		totals := make(map[string]uint)
		users := make(map[uint][]User)
		for _, user := range userlist {
			grade := user.GradeNow()
			if user.Grade == 0 || grade < 9 || grade > 12 {
				continue
			}
			users[grade] = append(users[grade], user)

			totals[user.Email] = database.Total(user.Email)
		}

		grades := make([]uint, 0, len(users))
		for grade := uint(9); grade <= 12; grade++ {
			studentlist, ok := users[grade]
			if !ok {
				continue
			}

			grades = append(grades, grade)
			sort.Slice(studentlist, func(i, j int) bool {
				return studentlist[i].Name < studentlist[j].Name
			})
		}

		return 200, "files/admin.html", map[string]interface{}{
			"User":     user,
			"Students": users,
			"Grades":   grades,
			"Totals":   totals,
		}
	}))

	// GET /all/flagged
	// Serves the Suspicious Entry list.
	r.Handle("/all/flagged", NewTemplateHandler(true, true, func(student string, user User, query url.Values, vars map[string]string) (uint16, string, interface{}) {
		flagged, err := database.Flagged()
		if err != nil {
			return 500, "", nil
		}

		users, err := database.Users()
		if err != nil {
			return 500, "", nil
		}

		return 200, "files/flagged.html", map[string]interface{}{
			"User":     user,
			"Students": users,
			"Entries":  flagged,
		}
	}))

	// GET /roster
	// Serves the Update Roster page.
	r.Handle("/roster", NewTemplateHandler(true, true, func(student string, user User, query url.Values, vars map[string]string) (uint16, string, interface{}) {
		return 200, "files/roster.html", map[string]interface{}{
			"User": user,
		}
	}))

	// GET /{email}
	// Lists entries.
	r.Handle("/{email}", NewTemplateHandler(true, false, func(student string, user User, query url.Values, vars map[string]string) (uint16, string, interface{}) {
		if student == "" {
			return 403, "", nil
		}

		entries, err := database.List(student)
		if err != nil {
			return 404, "", nil
		}

		keys := make(map[uint][]string)
		grades := []uint(nil)

		for key, entry := range entries {
			grade := user.GradeAt(entry.Date)
			keys[grade] = append(keys[grade], key)
		}

		for grade, keylist := range keys {
			grades = append(grades, grade)
			entries.SortKeys(keylist)
		}

		sort.Slice(grades, func(i, j int) bool {
			return grades[i] > grades[j]
		})

		return 200, "files/list.html", map[string]interface{}{
			"User":    user,
			"Student": database.User(student),
			"Entries": entries,
			"Keys":    keys,
			"Grades":  grades,
			"Total":   database.Total(student),
		}
	}))

	// GET /{email}/{key}
	// Views, edits, or adds a specific entry.
	r.Handle("/{email}/{key}", NewTemplateHandler(true, false, func(student string, user User, query url.Values, vars map[string]string) (uint16, string, interface{}) {
		if student == "" {
			return 403, "", nil
		}

		key := vars["key"]

		var entry *Entry
		if key == "add" {
			entry = EntryFromQuery(query)
		} else {
			var err error
			entry, err = database.Get(student, key)
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
			"Student": database.User(student),
			"Entry":   entry,
			"Key":     key,
			"Action":  action,
		}
	}))

	// GET /{email}/{key}/duplicate
	// Creates a new entry that is a replica of the old one, with the exception that the new entry's date is set to the current day.
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

		w.Header().Set("Location", "/"+email+"/add?"+entry.EncodeQuery().Encode())
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
