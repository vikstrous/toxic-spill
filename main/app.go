package main

import (
	"fmt"
	"log"
	"net/http"

	toxiproxy "github.com/Shopify/toxiproxy/client"
	"github.com/donhcd/dockerclient"
	"github.com/gorilla/mux"
)

var tp *toxiproxy.Client

func addProxyHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi")
}

func main() {
	dc, err := dockerclient.NewDockerClient("unix:///var/run/docker.sock", nil)
	if err != nil {
		log.Fatalf("Failed to init dockerclient: %v", err)
	}
	tp = toxiproxy.NewClient(getTpServer(dc) + ":8474")
	fs := http.FileServer(http.Dir("assets"))
	http.Handle("/assets", fs)

	r := mux.NewRouter()
	r.HandleFunc("/proxy", addProxyHandler).Methods("GET")

	log.Println("Listening...")
	http.ListenAndServe(":3000", r)
}

func getTpServer(dc dockerclient.Client) string {
	containers, err := dc.ListContainers(false, false, "")
	if err != nil {
		log.Fatalf("Failed to get docker containers list: %v", err)
	}
	for _, container := range containers {
		if container.Image == "shopify/toxiproxy:latest" {
			return container.Id
		}
	}
	log.Fatal("couldn't find a running toxiproxy")
	return ""
}
