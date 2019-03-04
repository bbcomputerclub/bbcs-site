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
)

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

type UserData struct {
	Name string
	Email string
}

func getUser(token string) (UserData, error) {
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + token)	
	if err != nil {
		return UserData{}, errors.New("Something went wrong. Try again.")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return UserData{}, errors.New("Something went wrong. Try again.")
	}

	data := make(map[string]interface{})
	json.Unmarshal(body, &data)

	if data["error"] != nil {
		return UserData{}, errors.New("Not signed in: " + fmt.Sprint(data["error"]))
	}

	if fmt.Sprint(data["hd"]) != "blindbrook.org" {
		return UserData{}, errors.New("That account isn't associated with Blind Brook.")
	}

	out := UserData{}
	out.Email = fmt.Sprint(data["email"])
	if out.Email != "bstarr@blindbrook.org" {
		out.Name = fmt.Sprint(data["name"])
	} else {
		out.Name = "Robert Starr"
	}

	return out, nil
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

	http.HandleFunc("/icons/", func (w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".png") {
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
			bodybyte, err := ioutil.ReadFile("icon.svg")
			if err != nil {
				w.Header().Set("Content-Type", "text/plain")			
				w.WriteHeader(500)
				return
			}
			body := string(bodybyte)
			len := r.URL.Path[7:len(r.URL.Path) - 4]

			if _, err := strconv.Atoi(len); err != nil {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(404)
				return
			}

			i := strings.Index(body, "width=\"")
			j := strings.Index(body[i+7:], "\"") + i+7
			body = body[0:i + 7] + len + body[j:]

			i = strings.Index(body, "height=\"")
			j = strings.Index(body[i+8:], "\"") + i+8
			body = body[0:i + 8] + len + body[j:]

			w.Header().Set("Content-Type", "image/svg+xml")
			w.Write([]byte(body))
			return
		}
		w.WriteHeader(404)
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

	http.HandleFunc("/manifest.json", func (w http.ResponseWriter, r *http.Request) { 
		body, err := ioutil.ReadFile("manifest.json")
		if err != nil {
			w.WriteHeader(500)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	})

	http.HandleFunc("/generator", func (w http.ResponseWriter, r *http.Request) {
		file, err := os.Open("generator.html")
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		io.Copy(w, file)
	})

	http.HandleFunc("/source", func (w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://github.com/bbcomputerclub/bbcs-site/")
		w.WriteHeader(301)
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

		if index >= 0 {
			if !DBGet(user.Email, index).Editable() {
				w.WriteHeader(403)
				return
			}
		}

		newEntry := entryFromQuery(query)
		if !newEntry.Editable() {
			w.WriteHeader(403)
			return		
		}
		
		DBSet(user.Email, newEntry, index)
	
		w.Header().Set("Location", "/list?token=" + query.Get("token"))
		w.WriteHeader(302)
	})

	http.HandleFunc("/delete", func (w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		user, err := getUser(query.Get("token"))
		if err != nil {
			w.WriteHeader(400)
			return
		}
		index, err := strconv.Atoi(query.Get("entry"))
		if err != nil || index < 0 {
			w.WriteHeader(400)
			return
		}

		if !DBGet(user.Email, index).Editable() {
			w.WriteHeader(403)
			return
		}

		DBRemove(user.Email, index)
	
		w.Header().Set("Location", "/list?token=" + query.Get("token"))
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
