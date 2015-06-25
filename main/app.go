package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	toxiproxy "github.com/Shopify/toxiproxy/client"
	"github.com/donhcd/dockerclient"
	"github.com/gorilla/mux"
)

var tp *toxiproxy.Client
var tpIp string
var dc dockerclient.Client
var firstAvailablePort uint64 = 9000

func addProxyHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var arg struct {
		Container string
		IPAddress string `json:"ipAddress"`
		Port      uint16
	}
	if err := json.NewDecoder(r.Body).Decode(&arg); err != nil {
		http.Error(w, "malformed request body", http.StatusBadRequest)
		log.Printf("bad request\n")
		return
	}

	var containerIP string
	if containerInfo, err := dc.InspectContainer(arg.Container); err != nil {
		log.Printf("can't talk to docker\n")
		http.Error(w, "can't talk to docker", http.StatusInternalServerError)
		return
	} else {
		containerIP = containerInfo.NetworkSettings.IPAddress
	}

	newTpPort := findNewTpPort()

	tpProxy := tp.NewProxy(&toxiproxy.Proxy{
		Name:     fmt.Sprintf("%s_%s_%d", arg.Container, arg.IPAddress, arg.Port),
		Listen:   fmt.Sprintf("%s:%d", tpIp, newTpPort),
		Upstream: fmt.Sprintf("%s:%d", arg.IPAddress, arg.Port),
		Enabled:  true,
	})
	if err := tpProxy.Create(); err != nil {
		log.Printf("can't create new tp proxy: %v\n", err)
		http.Error(w, "can't create new tp proxy", http.StatusInternalServerError)
		return
	}

	iptablesCmdString := fmt.Sprintf("iptables -t nat -I PREROUTING 1 -s %s -p tcp -d %s --dport %d -j DNAT --to-destination %s:%d", containerIP, arg.IPAddress, arg.Port, tpIp, newTpPort)
	iptablesCmdSlice := strings.Split(iptablesCmdString, " ")
	iptablesCmd := exec.Command(iptablesCmdSlice[0], iptablesCmdSlice[1:]...)
	iptablesCmd.Stdout = os.Stdout
	iptablesCmd.Stderr = os.Stderr
	if err := iptablesCmd.Run(); err != nil {
		log.Printf("failed to run iptables command: %v\n", err)
		http.Error(w, "can't iptables", http.StatusInternalServerError)
		return
	}
	log.Printf("successfully ran iptables command %q\n", iptablesCmdString)

	// TODO add proxy to list of proxies
	if err := json.NewEncoder(w).Encode(tpProxy); err != nil {
		log.Printf("failed to write tp proxy info: %v\n", err)
	}
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
	for i := 0; i < len(conns_arr); i += 2 {
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
	var err error
	dc, err = dockerclient.NewDockerClient("unix:///var/run/docker.sock", nil)
	if err != nil {
		log.Fatalf("Failed to init dockerclient: %v", err)
	}
	tpIp = getTpHost(dc)
	tp = toxiproxy.NewClient("http://" + tpIp + ":8474")

	proxies, err := tp.Proxies()
	if err != nil {
		log.Fatalf("Failed to list toxiproxy proxies: %v", err)
	}
	fmt.Printf("existing proxies: %v\n", proxies)

	fs := http.FileServer(http.Dir("assets"))

	r := mux.NewRouter()
	r.HandleFunc("/proxy", addProxyHandler).Methods("POST")
	r.PathPrefix("/").Handler(fs)

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

func findNewTpPort() uint64 {
	newPort := firstAvailablePort
	firstAvailablePort++
	return newPort
}
