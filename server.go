package main

/* Simple http server used for testing */

import (
	"net/http"
	"io/ioutil"
	"os"
	"mime"
	"strings"
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
		w.Write(bytes)
		w.WriteHeader(200)
	})
	http.ListenAndServe(":8080", nil)
}
