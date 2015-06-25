package main

import (
	"fmt"
	"io"
	"net/http"
)

type FileServer http.Dir

func (fs FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}
	f, err := ((http.Dir)(fs)).Open(path)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Oh shit: %v", err)
		return
	} else if stat, _ := f.Stat(); stat.IsDir() {
		w.WriteHeader(404)
		fmt.Fprint(w, "404 not found, bitches")
		return
	}
	defer f.Close()

	io.Copy(w, f)
}
