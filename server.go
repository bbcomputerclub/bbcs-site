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
	"encoding/json"
	"errors"
	"os"
)

type UserData struct {
	Name string
	Email string
}

func getUser(token string) (UserData, error) {
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + token)	
	if err != nil {
		return UserData{}, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return UserData{}, err
	}

	data := make(map[string]interface{})
	json.Unmarshal(body, &data)

	if data["error"] != nil {
		return UserData{}, errors.New(fmt.Sprint(data["error"]))
	}

	if fmt.Sprint(data["hd"]) != "blindbrook.org" {
		return UserData{}, errors.New("Not blindbrook")
	}

	return UserData{
				Name: fmt.Sprint(data["name"]),
				Email: fmt.Sprint(data["email"]),
			}, nil
}

func process(in []byte, query url.Values) ([]byte, error) {
	var re = regexp.MustCompile("(?s)\\[\\[.*?\\]\\]")
	user, err := getUser(query.Get("token"))
	if err != nil {
		return nil, err
	}

	var entry DBEntry
	index, err := strconv.Atoi(query.Get("entry"))
	if err != nil {
		index = -1
	}
	entry = DBGet(user.Email, index)
 	
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
			if cmd[1] == "total" {
				return []byte(fmt.Sprint(DBTotal(user.Email)))
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
			if cmd[1] == "action" {
				if query.Get("entry")[0] == '-' {
					return []byte("Add")
				} else {
					return []byte("Edit")
				}
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
	}), nil
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

		if res, err := process(body, r.URL.Query());  err == nil {
			w.Header().Set("Content-Type", "text/html")
			w.Write(res)
		} else {
			query := r.URL.Query()
			query.Del("token")
			w.Header().Set("Location", "/?list?" + query.Encode() + "#error:" + err.Error())
			w.WriteHeader(302)
		}
	})

	http.HandleFunc("/edit", func (w http.ResponseWriter, r *http.Request) { 
		/* TODO: Need to disable after 30 days */
	
		body, err := ioutil.ReadFile("edit.html")
		if err != nil {
			w.WriteHeader(500)
			return
		}
		
		if res, err := process(body, r.URL.Query());  err == nil {
			w.Header().Set("Content-Type", "text/html")
			w.Write(res)
		} else {
			query := r.URL.Query()
			query.Del("token")
			w.Header().Set("Location", "/?edit?" + query.Encode() + "#error:" + err.Error())
			fmt.Println(w.Header().Get("Location"))
			w.WriteHeader(302)
		}
	})

	http.HandleFunc("/add", func (w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		query.Set("entry", "-1")
		w.Header().Set("Location", "/edit?" + query.Encode())
		w.WriteHeader(302)
	})

	http.HandleFunc("/update", func (w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		user, err := getUser(query.Get("token"))
		if err != nil {
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
