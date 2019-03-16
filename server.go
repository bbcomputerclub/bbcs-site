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
	"os"
	"html/template"
	"math/rand"
)

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

type FileHandler string
type FileHandlerData struct {
	Entry *DBEntry	// Current entry, or nil
	EntryIndex int	// Index of entry
	
	User UserData	// Current logged-in user
	Student UserData // Which student he is looking at

	Students []UserData
}

const (
	ACTION_VIEW = "View"
	ACTION_EDIT = "Edit"
	ACTION_ADD = "Add"
)

func (d FileHandlerData) Action() string {
	switch {
	case d.EntryIndex < 0:
		return ACTION_ADD
	case !d.Entry.Editable() && !d.User.Admin:
		return ACTION_VIEW
	default:
		return ACTION_EDIT
	}
}

func (d FileHandlerData) StudentEntries() []*DBEntry {
	return DBList(d.Student.Email)
}

func (f FileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := dataFromRequest(r)
	if err != nil {
		w.Header().Set("Refresh", "0;url=/#error:" + url.QueryEscape(err.Error()))
		w.WriteHeader(400)
		return
	}

	t, err := template.New(string(f)).Funcs(template.FuncMap{
		"time": func (from int) time.Time {
			return time.Now().AddDate(0,0,from)
		},
	}).ParseFiles(string(f))
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(os.Stderr, err)
		return
	}
	err = t.Execute(w, data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return		
	}
}

func dataFromRequest(r *http.Request) (*FileHandlerData, error) {
	out := new(FileHandlerData)

	user, err := SignedUserHTTP(r)
	if err != nil {
		return nil, err
	}	

	query := r.URL.Query()

	out.User = user
	if user.Admin && len(query["user"]) != 0 {
		out.Student = UserFromEmail(query.Get("user"))
	} else {
		out.Student = user
	}

	if user.Admin {
		doc := DBDocumentGet()
		for email, _ := range doc {
			out.Students = append(out.Students, UserFromEmail(email))
		}
	}

	out.EntryIndex, err = strconv.Atoi(query.Get("entry"))
	if err == nil {
		if out.EntryIndex >= 0 {
			out.Entry = DBGet(out.Student.Email, out.EntryIndex)
		} else {
			out.Entry = entryFromQuery(query)
		}
	} else {
		out.EntryIndex = -1
	}

	return out, nil
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

	http.Handle("/source", http.RedirectHandler("https://github.com/bbcomputerclub/bbcs-site", 301))

	http.HandleFunc("/calendar", func (w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "calendar.html")
	})


	/* GET /signin
	 *
	 * Signs the user in.
	 * Parameters:
	 * 	  token: The token
	 *    redirect: An escaped URI
	 */
	http.HandleFunc("/signin", func (w http.ResponseWriter, r *http.Request) {
		user, err := UserFromToken(r.URL.Query().Get("token"))
		if err != nil {
			w.Header().Set("Refresh", "0; url=/#error:" + url.QueryEscape(err.Error()))
			w.WriteHeader(403)
			return
		}

		sid := Signin(user)
		http.SetCookie(w, &http.Cookie{Name:"BBCS_SESSION_ID", Value:sid, HttpOnly:true})

		redirect := r.URL.Query().Get("redirect")
		redirectEsc, err := url.QueryUnescape(redirect)
		if len(redirect) == 0 || err != nil {
			if user.Admin {
				w.Header().Set("Location", "/admin")
			} else {
				w.Header().Set("Location", "/list")			
			}
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
		Signout(sid)	

		http.SetCookie(w, &http.Cookie{Name:"BBCS_SESSION_ID",Value:"",MaxAge:-1})
		w.Header().Set("Location", "/#signout")
		w.WriteHeader(303)
	})
	
	http.Handle("/list", FileHandler("list.html"))
	http.Handle("/admin", FileHandler("admin.html"))
	http.Handle("/edit", FileHandler("edit.html"))

	http.HandleFunc("/add", func (w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		query.Set("entry", "-1")
		w.Header().Set("Location", "/edit?" + query.Encode())
		w.WriteHeader(302)
	})

	http.HandleFunc("/duplicate", func (w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		user, err := SignedUserHTTP(r)
		if err != nil {
			w.WriteHeader(400)
		}
		student := user
		if user.Admin && len(query["user"]) != 0 {
			student = UserFromEmail(query.Get("user"))
		}

		// Get entry
		index, err := strconv.Atoi(query.Get("entry"))
		if err != nil {
			w.WriteHeader(400)
			return
		}
		
		entry := DBGet(student.Email, index)
		if entry == nil {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(400)
			io.WriteString(w, "Entry #" + r.URL.Query().Get("entry") + " does not exist")		
			return
		}

		newQuery := entryToQuery(entry)
		newQuery.Set("entry", "-1")
		newQuery.Set("user", student.Email)
		w.Header().Set("Location", "/edit?" + newQuery.Encode())
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

		user, err := SignedUserHTTP(r)
		if err != nil {
			w.WriteHeader(403)
			return
		}

		student := user
		if user.Admin && len(query["user"]) != 0 {
			student = UserFromEmail(query.Get("user"))
		}

		// Get entry
		index, err := strconv.Atoi(query.Get("entry"))
		if err != nil {
			w.WriteHeader(400)
			return
		}

		newEntry := entryFromQuery(query)

		// Make sure entry is editable
		if !user.CanEdit(DBGet(student.Email, index)) || !user.CanEdit(newEntry) {
			w.WriteHeader(403)
			return
		}

		// Make changes
		DBSet(student.Email, newEntry, index)

		// Redirect
		w.Header().Set("Location", "/list?user=" + student.Email)
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

		user, err := SignedUserHTTP(r)
		if err != nil {
			w.WriteHeader(403)
			return
		}
		student := user
		if user.Admin && len(query["user"]) != 0 {
			student = UserFromEmail(query.Get("user"))
		}

		// Get entry
		index, err := strconv.Atoi(query.Get("entry"))
		if err != nil {
			w.WriteHeader(400)
			return
		}

		// Make sure existing entry is editable
		if !user.CanEdit(DBGet(student.Email, index)) {
			w.WriteHeader(403)
			return
		}

		// Make changes
		DBRemove(student.Email, index)

		// Redirect
		w.Header().Set("Location", "/list")
		w.WriteHeader(302)
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
