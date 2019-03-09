package main

import (
	"net/http"
	"net/url"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"strconv"
	"time"
	"encoding/json"
	"errors"
	"os"
	"math/rand"
)

const (
	ACTION_VIEW = "View"
	ACTION_EDIT = "Edit"
	ACTION_ADD = "Add"
)

type UserData struct {
	Name string
	Email string
}

/* Creates an entry from url parameters */
func entryFromQuery(query url.Values) *DBEntry {
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

func entryToQuery(entry *DBEntry) url.Values {
	out := url.Values{}
	out.Set("name", entry.Name)
	out.Set("hours", strconv.FormatUint(uint64(entry.Hours), 10))
	out.Set("date", entry.Date.Format("2006-01-02"))
	out.Set("org", entry.Organization)
	out.Set("contactname", entry.ContactName)
	out.Set("contactemail", entry.ContactEmail)
	out.Set("contactphone", strconv.FormatUint(uint64(entry.ContactPhone), 10))
	return out
}

/* Returns stuff needed for `process()` */
func dataFromQuery(query url.Values, sid string) (UserData, *DBEntry, int) {
	user, ok := signinMap[sid]
	if !ok {
		return UserData{}, nil, -1
	}

	var entry *DBEntry = nil
	var entryIndex int
	var err error
	if entryIndex, err = strconv.Atoi(query.Get("entry")); err == nil && entryIndex >= 0 {
		entry = DBGet(user.Email, entryIndex)
	} else {
		entry = entryFromQuery(query)
		entryIndex = -1
	}	

	return user, entry, entryIndex
}

/* Passes token through Google servers to validate it */
func getUser(token string) (UserData, error) {
	// Next 8 lines: Retrieves data from Google servers
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + token)	
	if err != nil {
		return UserData{}, errors.New("Something went wrong. Try again.")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return UserData{}, errors.New("Something went wrong. Try again.")
	}

	// The data is stored in JSON. Unmarshal the data
	data := make(map[string]interface{})
	json.Unmarshal(body, &data)

	// If the error field is present, there is an error
	if data["error"] != nil {
		return UserData{}, errors.New("Not signed in: " + fmt.Sprint(data["error"]))
	}

	// Make sure the domain is Blind Brook (the account is from Blind Brook)
	if fmt.Sprint(data["hd"]) != "blindbrook.org" {
		return UserData{}, errors.New("That account isn't associated with Blind Brook.")
	}

	// Create UserData struct; fmt.Sprint converts things to strings (just in case it's not a string)
	out := UserData{}
	out.Email = fmt.Sprint(data["email"])
	if out.Email != "bstarr@blindbrook.org" {
		out.Name = fmt.Sprint(data["name"])
	} else {
		out.Name = "Robert Starr"
	}

	return out, nil
}

/* Maps session IDs to tokens */
var signinMap = make(map[string]UserData)
/* Generates a session ID */
func signinGen() string {
	out := strconv.FormatInt(rand.Int63(), 36)
	if _, ok := signinMap[out]; ok { // if seesion id already is a thing
		return signinGen() // try again
	}
	return out
}

func main() {
	rand.Seed(time.Now().UnixNano())

	/* Static pages */
	http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(404)
			return
		}
	
		http.ServeFile(w, r, "login.html")
	})

	http.HandleFunc("/icons/", func (w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".png") {
		/* PNG
		 * PNG files are stored in the directory as icon-N.png
		 * This section simply retrieves the file and serves it
		 */
			file, err := os.Open("icon-" + r.URL.Path[7:])
			if err != nil {
				w.Header().Set("Content-Type", "text/plain")					
				w.WriteHeader(404)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			io.Copy(w, file)	
			return
		} else if strings.HasSuffix(r.URL.Path, ".svg") {
		/* SVG
		 * SVG code is found in icon.svg
		 * This section retrieves icon.svg, replaces the width and height attributes, and then serves the modified file
		 */
			// Get icon.svg
			bodybyte, err := ioutil.ReadFile("icon.svg")
			if err != nil {
				w.Header().Set("Content-Type", "text/plain")			
				w.WriteHeader(500)
				return
			}
			body := string(bodybyte)
			len := r.URL.Path[7:len(r.URL.Path) - 4]

			// Make sure filename is an integer
			if _, err := strconv.Atoi(len); err != nil {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(404)
				return
			}

			// Replace width and height attributes
			i := strings.Index(body, "width=\"")
			j := strings.Index(body[i+7:], "\"") + i+7
			body = body[0:i + 7] + len + body[j:]

			i = strings.Index(body, "height=\"")
			j = strings.Index(body[i+8:], "\"") + i+8
			body = body[0:i + 8] + len + body[j:]

			// Serve
			w.Header().Set("Content-Type", "image/svg+xml")
			w.Write([]byte(body))
			return
		}
		w.WriteHeader(404)
	})
	
	http.HandleFunc("/style.css", func (w http.ResponseWriter, r *http.Request) { 
		http.ServeFile(w, r, "style.css")
	})

	http.HandleFunc("/manifest.json", func (w http.ResponseWriter, r *http.Request) { 
		http.ServeFile(w, r, "manifest.json")
	})

	http.HandleFunc("/generator", func (w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "generator.html")
	})

	http.HandleFunc("/source", func (w http.ResponseWriter, r *http.Request) {
		// Redirect to the GitHub repo; 301 is a permanant redirect
		w.Header().Set("Location", "https://github.com/bbcomputerclub/bbcs-site/")
		w.WriteHeader(301)
	})


	/* GET /signin
	 *
	 * Signs the user in.
	 * Parameters:
	 * 	  token: The token
	 *    redirect: An escaped URI
	 */
	http.HandleFunc("/signin", func (w http.ResponseWriter, r *http.Request) {
		sid := signinGen()

		user, err := getUser(r.URL.Query().Get("token"))
		if err != nil {
			w.Header().Set("Refresh", "0; url=/#error:" + url.QueryEscape(err.Error()))
			w.WriteHeader(403)
			return
		}

		signinMap[sid] = user

		redirect := r.URL.Query().Get("redirect")
		if redirect == "" {
			redirect = "/list"
		}

		http.SetCookie(w, &http.Cookie{Name:"BBCS_SESSION_ID", Value:sid, HttpOnly:true})

		redirectEsc, err := url.QueryUnescape(redirect)
		if err != nil {
			w.Header().Set("Location", "/list")
		} else {
			w.Header().Set("Location", redirectEsc)
		}
		w.WriteHeader(303)
	})

	http.HandleFunc("/signout", func (w http.ResponseWriter, r *http.Request) {
		sidC, err := r.Cookie("BBCS_SESSION_ID")
		if err != nil {
			w.WriteHeader(400)
			return			
		}
		sid := sidC.Value
	
		delete(signinMap, sid)

		http.SetCookie(w, &http.Cookie{Name:"BBCS_SESSION_ID",Value:"",MaxAge:-1})
		w.Header().Set("Location", "/#signout")
		w.WriteHeader(303)
	})
	
	http.HandleFunc("/list", func (w http.ResponseWriter, r *http.Request) { 
		sidC, err := r.Cookie("BBCS_SESSION_ID")
		if err != nil {
			w.Header().Set("Location", "/#error:Not%20signed%20in")		
			w.WriteHeader(303)
			return
		}
		sid := sidC.Value

		body, err := ioutil.ReadFile("list.html")
		if err != nil {
			w.WriteHeader(500)
			return
		}

		user, entry, entryIndex := dataFromQuery(r.URL.Query(), sid)
		if user.Email == "" {
			w.Header().Set("Location", "/#error:Not%20signed%20in")		
			w.WriteHeader(303)
			return
		}
		if res, err := process(body, user, entry, entryIndex);  err == nil {
			w.Header().Set("Content-Type", "text/html")
			w.Write(res)
		} else {
			// If there was an error, redirect to the login page
			w.Header().Set("Location", "/#error:" + url.QueryEscape(err.Error()))
			w.WriteHeader(303)
		}
	})

	http.HandleFunc("/edit", func (w http.ResponseWriter, r *http.Request) {
		sidC, err := r.Cookie("BBCS_SESSION_ID")
		if err != nil { 
			w.Header().Set("Location", "/?%2Fedit%3F" + url.QueryEscape(r.URL.Query().Encode()) + "#error:Not%20signed%20in")		
			w.WriteHeader(303)
			return
		}
		sid := sidC.Value

		body, err := ioutil.ReadFile("edit.html")
		if err != nil {
			w.WriteHeader(500)
			return
		}
		
		user, entry, entryIndex := dataFromQuery(r.URL.Query(), sid)
		if user.Email == "" {
			w.Header().Set("Location", "/?%2Fedit%3F" + url.QueryEscape(r.URL.Query().Encode()) + "#error:Not%20signed%20in")
			w.WriteHeader(303)			
			return
		}
		if res, err := process(body, user, entry, entryIndex);  err == nil {
			w.Header().Set("Content-Type", "text/html")
			w.Write(res)
		} else {
			// If there was an error, redirect to the login page
			w.Header().Set("Location", "/?%2Fedit%3F" + url.QueryEscape(r.URL.Query().Encode()) + "#error:" + url.QueryEscape(err.Error()))
			w.WriteHeader(303)
		}
	})

	http.HandleFunc("/add", func (w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		query.Set("entry", "-1")
		w.Header().Set("Location", "/edit?" + query.Encode())
		w.WriteHeader(302)
	})

	http.HandleFunc("/duplicate", func (w http.ResponseWriter, r *http.Request) {
		sidC, err := r.Cookie("BBCS_SESSION_ID")
		if err != nil {
			w.Header().Set("Refresh", "0;url=/#error:Not%20signed%20in")		
			w.WriteHeader(403)
			return
		}
		sid := sidC.Value
	
		user, ok := signinMap[sid]
		if !ok {
			w.Header().Set("Refresh", "0;url=/#error:Not%20signed%20in")
			w.WriteHeader(403)
			return
		}

		entryIndex, err := strconv.Atoi(r.URL.Query().Get("entry"))
		if err != nil {
			w.WriteHeader(400)
			return
		}
		entry := DBGet(user.Email, entryIndex)
		if entry == nil {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(400)
			io.WriteString(w, "Entry #" + r.URL.Query().Get("entry") + " does not exist")		
			return
		}

		query := entryToQuery(entry)
		query.Set("entry", "-1")
		w.Header().Set("Location", "/edit?" + query.Encode())
		w.WriteHeader(303)
	});

	// Updates an entry
	http.HandleFunc("/update", func (w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" && r.Method != "PUT" {
			w.Header().Set("Allow", "POST, PUT")
			w.WriteHeader(405)
			return			
		}
	
		r.ParseForm()
		query := r.PostForm

		sidC, err := r.Cookie("BBCS_SESSION_ID")
		if err != nil {
			w.WriteHeader(403)
			return
		}
		sid := sidC.Value
		
		user, ok := signinMap[sid]
		if !ok {
			w.WriteHeader(403)
			return
		}
		index, err := strconv.Atoi(query.Get("entry"))
		if err != nil {
			w.WriteHeader(400)
			return
		}

		// Make sure existing entry (if there is one) is editable
		if index >= 0 {
			if !DBGet(user.Email, index).Editable() {
				w.WriteHeader(403)
				return
			}
		}

		newEntry := entryFromQuery(query)
		if !newEntry.Editable() { // Make sure new entry is editable (not past 30 days)
			w.WriteHeader(403)
			return		
		}

		// Make changes
		DBSet(user.Email, newEntry, index)

		// Redirect
		w.Header().Set("Location", "/list")
		w.WriteHeader(302)
	})

	// Removes an entry
	http.HandleFunc("/delete", func (w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" && r.Method != "DELETE" {
			w.Header().Set("Allow", "POST, DELETE")
			w.WriteHeader(405)
			return			
		}

		r.ParseForm()
		query := r.PostForm
	
		sidC, err := r.Cookie("BBCS_SESSION_ID")
		if err != nil {
			w.WriteHeader(403)
			return
		}
		sid := sidC.Value

		user, ok := signinMap[sid]
		if !ok {
			w.WriteHeader(403)
			return
		}
		index, err := strconv.Atoi(query.Get("entry"))
		if err != nil || index < 0 {
			w.WriteHeader(400)
			return
		}

		// Makes sure entry is editable
		if !DBGet(user.Email, index).Editable() {
			w.WriteHeader(403)
			return
		}

		DBRemove(user.Email, index)
	
		w.Header().Set("Location", "/list")
		w.WriteHeader(303)
	})

	port := uint64(0)
	var err error = nil
	switch len(os.Args) {
	case 0,1:
		port = 8080
	case 2:
		port, err = strconv.ParseUint(os.Args[1], 10, 64)
		if err == nil {
			break
		}
		fallthrough
	default:
		fmt.Fprintf(os.Stderr, "usage: %s [port]", os.Args[0])
	}

	fmt.Printf("http://localhost:%v/\n", port)
	err = http.ListenAndServe(":" + fmt.Sprint(port), nil)
	fmt.Fprintln(os.Stderr, err)
}
