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
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/client"
	"github.com/donhcd/dockerclient"
	"github.com/gorilla/mux"
)

var firstAvailablePort uint64 = 9000

type containerProxyInfo struct {
	Name    string            `json:"name"`
	Proxies []toxiproxy.Proxy `json:"proxies"`
}

var containerProxies = []containerProxyInfo{
	{
		Name: "backstabbing_sinoussi",
		Proxies: []toxiproxy.Proxy{
			{
				Name:     "derp",
				Upstream: "google.com:80",
			},
		},
	},
	{
		Name: "gloomy_pasteur",
	},
}

func (s *Server) addProxyHandler(w http.ResponseWriter, r *http.Request) {
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
	if containerInfo, err := s.dc.InspectContainer(arg.Container); err != nil {
		log.Printf("can't talk to docker\n")
		http.Error(w, "can't talk to docker", http.StatusInternalServerError)
		return
	} else {
		containerIP = containerInfo.NetworkSettings.IPAddress
	}

	newTpPort := findNewTpPort()

	tpProxy := s.tp.NewProxy(&toxiproxy.Proxy{
		Name:     fmt.Sprintf("%s_%s_%d", arg.Container, arg.IPAddress, arg.Port),
		Listen:   fmt.Sprintf("%s:%d", s.tpIP, newTpPort),
		Upstream: fmt.Sprintf("%s:%d", arg.IPAddress, arg.Port),
		Enabled:  true,
	})
	if err := tpProxy.Create(); err != nil {
		log.Printf("can't create new tp proxy: %v\n", err)
		http.Error(w, "can't create new tp proxy", http.StatusInternalServerError)
		return
	}

	iptablesCmdString := fmt.Sprintf("iptables -t nat -I PREROUTING 1 -s %s -p tcp -d %s --dport %d -j DNAT --to-destination %s:%d", containerIP, arg.IPAddress, arg.Port, s.tpIP, newTpPort)
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

type Conn struct {
	SrcIp   string
	SrcPort string
	DstIp   string
	DstPort string
}
type ConnCache struct {
	conn      Conn
	last_seen time.Time
}

type Server struct {
	queryConns      chan bool
	queryConnsReply chan []Conn
	tp              *toxiproxy.Client
	tpIP            string
	dc              dockerclient.Client
}

func (s *Server) connsHandler(w http.ResponseWriter, r *http.Request) {
	s.queryConns <- true
	reply := <-s.queryConnsReply
	//log.Println("webserver")
	//log.Println(reply)
	json.NewEncoder(w).Encode(reply)
}

func getProxiesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(containerProxies)
}

func getActiveConns() []Conn {
	iftop_out, err := exec.Command("iftop", "-n", "-i", "docker0", "-P", "-N", "-t", "-s", "1").Output()
	if err != nil {
		log.Printf("Failed to execute iftop")
		return nil
	}

	results := []Conn{}

	re := regexp.MustCompile("(\\d+(\\.\\d+){3}):(\\d+)")
	conns := strings.Split(fmt.Sprintf("%s", iftop_out), "\n--------------------------------------------------------------------------------------------\n")[1]
	// when there are no entries there are two lines of --------- one after another
	if conns[0] == '-' {
		return nil
	}
	conns_arr := strings.Split(conns, "\n")
	for i := 0; i < len(conns_arr); i += 2 {
		ip_port_dst := re.FindStringSubmatch(conns_arr[i])
		ip_port_src := re.FindStringSubmatch(conns_arr[i+1])
		conn := Conn{ip_port_src[1], ip_port_src[3], ip_port_dst[1], ip_port_dst[3]}
		//log.Printf("%s connecting to %s on port %s", conn.src_ip, conn.dst_ip, conn.dst_port)
		results = append(results, conn)
	}
	return results
}

func connPoller(c chan Conn) {
	for {
		conns := getActiveConns()
		log.Println("poller")
		log.Println(conns)
		for _, conn := range conns {
			c <- conn
		}
	}
}

func connStateTracker(c chan Conn, query chan bool, reply chan []Conn) {
	conns := []ConnCache{}
	for {
		select {
		case conn := <-c:
			log.Println("state")
			log.Println(conn)
			conns = append(conns, ConnCache{conn, time.Now()})
		case <-query:
			// expire old entries
			new_list := []ConnCache{}
			for _, c := range conns {
				if !c.last_seen.Before(time.Now().Add(-30 * time.Second)) {
					new_list = append(new_list, c)
				}
			}
			conns = new_list

			// return new entries left
			ret := make([]Conn, len(conns))
			for i, c := range conns {
				ret[i] = c.conn
			}
			reply <- ret
		}
	}
}

func main() {
	dc, err := dockerclient.NewDockerClient("unix:///var/run/docker.sock", nil)
	if err != nil {
		log.Fatalf("Failed to init dockerclient: %v", err)
	}
	tpIP := getTpHost(dc)
	tp := toxiproxy.NewClient("http://" + tpIP + ":8474")

	proxies, err := tp.Proxies()
	if err != nil {
		log.Fatalf("Failed to list toxiproxy proxies: %v", err)
	}
	fmt.Printf("existing proxies: %v\n", proxies)

	fs := FileServer(http.Dir("assets"))

	s := Server{
		queryConns:      make(chan bool),
		queryConnsReply: make(chan []Conn),
		tp:              tp,
		tpIP:            tpIP,
		dc:              dc,
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/proxy", s.addProxyHandler).Methods("POST")
	r.HandleFunc("/api/proxies", getProxiesHandler).Methods("GET")
	r.HandleFunc("/api/conns", s.connsHandler).Methods("GET")
	r.PathPrefix("/").Handler(fs)

	// set up the channels for the gorouties
	recordConn := make(chan Conn)

	// start the poller and the state tracker
	go connPoller(recordConn)
	go connStateTracker(recordConn, s.queryConns, s.queryConnsReply)

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
