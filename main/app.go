package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strings"

	toxiproxy "github.com/Shopify/toxiproxy/client"
	"github.com/donhcd/dockerclient"
	"github.com/gorilla/mux"
)

var tp *toxiproxy.Client

func addProxyHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi")
}

func getActiveConns() [][]string {
	iftop_out, err := exec.Command("iftop", "-n", "-i", "docker0", "-P", "-N", "-t", "-s", "1").Output()
	if err != nil {
		log.Printf("Failed to execute iftop")
		return nil
	}

	results := [][]string{}

	re := regexp.MustCompile("(\\d+(\\.\\d+){3}):(\\d+)")
	conns := strings.Split(fmt.Sprintf("%s", iftop_out), "\n--------------------------------------------------------------------------------------------\n")[1]
	conns_arr := strings.Split(conns, "\n")
	for i := 0; i < len(conns_arr); i+=2 {
		ip_port_dst := re.FindStringSubmatch(conns_arr[i])
		ip_dst := ip_port_dst[1]
		port_dst := ip_port_dst[3]
		ip_port_src := re.FindStringSubmatch(conns_arr[i+1])
		ip_src := ip_port_src[1]
		//port_src := ip_port_src[3]
		//log.Printf("%s connecting to %s on port %s", ip_src, ip_dst, port_dst)
		results = append(results, []string{ip_src, ip_dst, port_dst})
	}
	return results
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

	log.Println(getActiveConns())

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
