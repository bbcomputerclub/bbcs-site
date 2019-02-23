package main

/* Simple http server used for testing */

import (
	"net/http"
	"io/ioutil"
	"os"
	"mime"
	"strings"
	"regexp"
	"time"
)

func main() {
	http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
		f, err := os.Open("." + r.URL.Path)
		if err != nil {
			w.WriteHeader(404)
			w.Write([]byte("Error 404: " + err.Error()))
			return
		}
		bytes, err := ioutil.ReadAll(f)
		if err != nil {
			w.WriteHeader(400)		
			w.Write([]byte("Error 400: " + err.Error()))
			return
		}

		i := strings.LastIndex(r.URL.Path, ".")
		if i != -1 {
			ext := r.URL.Path[i:]
			w.Header().Set("Content-Type", mime.TypeByExtension(ext))
		}

		resp := string(bytes)
		values := r.URL.Query()
		for key, _ := range values {
			resp = strings.Replace(resp, "[[" + key + "]]", values.Get(key), -1)
		}

		resp = strings.Replace(resp, "[[*today]]", time.Now().Format("2006-01-02"), -1)
		resp = regexp.MustCompile("\\[\\[.*\\]\\]").ReplaceAllString(resp, "")
	
		w.WriteHeader(200)		
		w.Write([]byte(resp))
	})
	http.ListenAndServe(":8080", nil)
}
