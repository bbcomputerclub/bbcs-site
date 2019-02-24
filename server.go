package main

import (
	"net/http"
	"net/url"
	"fmt"
	"io/ioutil"
	"strings"
	"regexp"
	"strconv"
	"time"
)

type UserData struct {
	Name string
	Email string
}

func getUser(_ string) UserData {
	// TODO
	return UserData{Name: "Bob", Email: "bob@example.com"}
}

func process(in []byte, query url.Values) []byte {
	var re = regexp.MustCompile("(?s)\\[\\[.*?\\]\\]")
	var user = getUser(query.Get("token"))
	if len(user.Email) == 0 {
		return nil
	}

	var entry DBEntry
	if len(query["entry"]) != 0 {
		index, err := strconv.Atoi(query.Get("entry"))
		if err != nil {
			index = -1
		}
		entry = DBGet(user.Email, index)
	}
 	
	return re.ReplaceAllFunc(in, func (rawcode []byte) []byte {
		code := string(rawcode[2:len(rawcode)-2])
		cmd := strings.Fields(code)
		if len(cmd) < 1 {
			return nil
		}
		switch cmd[0] {
		case "user":
			if len(cmd) != 2 { return nil }
			if cmd[1] == "name" {
				return []byte(user.Name)
			}
			if cmd[1] == "email" {
				return []byte(user.Email)
			}
			if cmd[1] == "token" {
				return []byte(query.Get("token"))
			}
			return nil
		case "entry":
			if len(cmd) != 2 { return nil }
			if cmd[1] == "name" { 
				return []byte(entry.Name)
			}
			if cmd[1] == "index" { 
				return []byte(query.Get("entry"))
			}
			if cmd[1] == "hours" { 
				return []byte(fmt.Sprint(entry.Hours))
			}
			if cmd[1] == "date" { 
				return []byte(entry.Date.Format("2006-01-02"))
			}
			if cmd[1] == "org" { 
				return []byte(entry.Organization)
			}
			if cmd[1] == "contact.name" { 
				return []byte(entry.ContactName)
			}
			if cmd[1] == "contact.email" { 
				return []byte(entry.ContactEmail)
			}
			return nil
		case "repeat":
			html := strings.Trim(code[6:], " \t\n")
			out := ""
			for i, entry := range DBList(user.Email) {
				out += strings.NewReplacer("[index]", fmt.Sprint(i), "[name]", entry.Name, "[token]", query.Get("token"), "[hours]", strconv.FormatUint(uint64(entry.Hours), 10)).Replace(html)
			}
			return []byte(out)
		default:
			return nil				
		}
	})	
}

func main() {
	http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(404)
			return
		}
	
		body, err := ioutil.ReadFile("login.html")
		if err != nil {
			w.WriteHeader(500)
			return
		}
		
		w.Header().Set("Content-Type", "text/html")
		w.Write(body)
	})
	
	http.HandleFunc("/style.css", func (w http.ResponseWriter, r *http.Request) { 
		body, err := ioutil.ReadFile("style.css")
		if err != nil {
			w.WriteHeader(500)
			return
		}
		
		w.Header().Set("Content-Type", "text/css")
		w.Write(body)
	})
	http.HandleFunc("/list", func (w http.ResponseWriter, r *http.Request) { 
		body, err := ioutil.ReadFile("list.html")
		if err != nil {
			w.WriteHeader(500)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write(process(body, r.URL.Query()))
	})

	http.HandleFunc("/edit", func (w http.ResponseWriter, r *http.Request) { 
		body, err := ioutil.ReadFile("edit.html")
		if err != nil {
			w.WriteHeader(500)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write(process(body, r.URL.Query()))
	})

	http.HandleFunc("/add", func (w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/edit?entry=-1&token=" + r.URL.Query().Get("token"))
		w.WriteHeader(302)
	})

	http.HandleFunc("/update", func (w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		user := getUser(query.Get("Token"))
		if len(user.Email) == 0 {
			w.WriteHeader(400)
			return
		}
		index, err := strconv.Atoi(query.Get("entry"))
		if err != nil {
			w.WriteHeader(400)
			return
		}
		hours, err := strconv.ParseUint(query.Get("hours"), 10, 64)
		if err != nil {
			hours = 1
		}
		date, err := time.Parse("2006-01-02", query.Get("date"))
		if err != nil {
			date = time.Now()
		}
		DBSet(user.Email, DBEntry{
			Name: query.Get("name"),
			Hours: uint(hours),
			Date: date,
			Organization: query.Get("org"),
			ContactName: query.Get("contactname"),
			ContactEmail: query.Get("contactemail"),
		}, index)
	
		w.Header().Set("Location", "/list?token=" + query.Get("token"))
		w.WriteHeader(302)
	})

	fmt.Println("http://localhost:8080/");
	http.ListenAndServe(":8080", nil)
}
