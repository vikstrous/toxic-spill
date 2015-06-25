package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

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
	tp = toxiproxy.NewClient("http://" + getTpHost(dc) + ":8474")

	proxies, err := tp.Proxies()
	if err != nil {
		log.Fatalf("Failed to list toxiproxy proxies: %v", err)
	}
	fmt.Printf("existing proxies: %v\n", proxies)

	fs := http.FileServer(http.Dir("assets"))

	r := mux.NewRouter()
	r.PathPrefix("/").Handler(fs)
	r.HandleFunc("/proxy", addProxyHandler).Methods("GET")

	log.Println("Listening on 3000...")
	http.ListenAndServe(":3000", r)
}

func getTpHost(dc dockerclient.Client) string {
	containers, err := dc.ListContainers(false, false, "")
	if err != nil {
		log.Fatalf("Failed to get docker containers list: %v", err)
	}
	for _, container := range containers {
		if strings.HasPrefix(container.Image, "shopify/toxiproxy") {
			if containerInfo, err := dc.InspectContainer(container.Id); err != nil {
				log.Fatalf("Failed to inspect container %s: %v", container.Id, err)
			} else {
				return containerInfo.NetworkSettings.IPAddress
			}
		}
	}
	log.Fatal("couldn't find a running toxiproxy")
	return ""
}
