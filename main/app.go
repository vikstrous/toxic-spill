package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func addProxyHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi")
}

func main() {
	fs := http.FileServer(http.Dir("assets"))
	http.Handle("/assets", fs)

	r := mux.NewRouter()
	r.HandleFunc("/proxy", addProxyHandler).Methods("GET")

	log.Println("Listening...")
	http.ListenAndServe(":3000", r)
}
