package main

import (
	"net/http"
	"net/url"
	"fmt"
	"io/ioutil"
	"strings"
	"regexp"
)

type UserData struct {
	Name string
	Email string
}

func getUser(query url.Values) UserData {
	token := query.Get("token")
	fmt.Println(token)
	// todo
	return UserData{Name: "Bob", Email: "bob@example.com"}
}

func process(in []byte, query url.Values) []byte {
	var re = regexp.MustCompile("(?s)\\[\\[.*?\\]\\]")
	var user  = UserData{}
	return re.ReplaceAllFunc(in, func (rawcode []byte) []byte {
		if len(user.Email) == 0 {
			user = getUser(query)
			if len(user.Email) == 0 {
				return nil
			}
		}
	
		code := string(rawcode[2:len(rawcode)-2])
		cmd := strings.Split(code, " ")
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
			return nil
		case "entry":
			if len(cmd) != 2 { return nil }
			
			if cmd[1] == "name" { 
			}
			return nil
		case "repeat":
			html := strings.Trim(code[6:], " \t\n")
			out := ""
			for i, entry := range DBList(user.Email) {
				out += strings.NewReplacer("[index]", fmt.Sprint(i), "[name]", entry.Name).Replace(html)
			}
			return nil
		default:
			return nil				
		}
	})	
}

func main() {
	http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("http://localhost:8080/");
	http.ListenAndServe(":8080", nil)
}
