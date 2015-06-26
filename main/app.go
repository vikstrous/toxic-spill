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
		Name:     fmt.Sprintf("%s;%s:%d", arg.Container, arg.IPAddress, arg.Port),
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
		tpProxy.Delete()
		return
	}
	log.Printf("successfully ran iptables command %q\n", iptablesCmdString)
	s.tpProxies[tpProxy.Name] = tpProxy

	if err := json.NewEncoder(w).Encode(tpProxy); err != nil {
		log.Printf("failed to write tp proxy info: %v\n", err)
	}
}

func (s *Server) deleteProxyHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var arg struct {
		Name string
	}
	if err := json.NewDecoder(r.Body).Decode(&arg); err != nil {
		http.Error(w, "malformed request body", http.StatusBadRequest)
		log.Println("bad request")
		return
	}

	if tpProxy, ok := s.tpProxies[arg.Name]; !ok {
		http.Error(w, "invalid proxy name", http.StatusBadRequest)
		log.Println("proxy doesn't exist")
		return
	} else if err := tpProxy.Delete(); err != nil {
		log.Printf("can't delete tp proxy: %v\n", err)
		http.Error(w, "can't delete tp proxy", http.StatusInternalServerError)
		return
	}

	delete(s.tpProxies, arg.Name)
}

func (s *Server) createToxicHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var arg struct {
		ToxicName string // latency, down, bandwidth, slow_close, timeout
		Upstream  bool
		Toxic     map[string]interface{} // https://github.com/Shopify/toxiproxy#toxics
	}
	if err := json.NewDecoder(r.Body).Decode(&arg); err != nil {
		http.Error(w, "malformed request body", http.StatusBadRequest)
		log.Println("bad request")
		return
	}

	proxyName := mux.Vars(r)["proxyName"]
	proxy, ok := s.tpProxies[proxyName]
	if !ok {
		http.Error(w, "no such proxy", http.StatusBadRequest)
		log.Println("no such proxy")
		return
	}
	direction := "downstream"
	if arg.Upstream {
		direction = "upstream"
	}
	if toxic, err := proxy.SetToxic(arg.ToxicName, direction, arg.Toxic); err != nil {
		http.Error(w, "failed to create toxic", http.StatusInternalServerError)
		log.Printf("failed to create toxic: %v\n", err)
		return
	} else {
		if arg.Upstream {
			proxy.ToxicsUpstream[arg.ToxicName] = toxic
		} else {
			proxy.ToxicsDownstream[arg.ToxicName] = toxic
		}
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
	tpProxies       map[string]*toxiproxy.Proxy
}

func (s *Server) getConnsHandler(w http.ResponseWriter, r *http.Request) {
	s.queryConns <- true
	reply := <-s.queryConnsReply
	//log.Println("webserver")
	//log.Println(reply)
	json.NewEncoder(w).Encode(reply)
}

func (s *Server) getProxiesHandler(w http.ResponseWriter, r *http.Request) {
	containerProxyMap := make(map[string][]*toxiproxy.Proxy)

	containers, err := s.dc.ListContainers(false, false, "")
	if err != nil {
		http.Error(w, "failed to load container list", http.StatusInternalServerError)
		log.Println("failed to load container list")
	}

	for _, container := range containers {
		containerProxyMap[canonicalName(container)] = []*toxiproxy.Proxy{}
	}

	for name, proxy := range s.tpProxies {
		containerName := strings.Split(name, ";")[0]
		containerProxyMap[containerName] = append(containerProxyMap[containerName], proxy)
	}

	type containerProxyInfo struct {
		Name    string             `json:"name"`
		Proxies []*toxiproxy.Proxy `json:"proxies"`
	}
	var containerProxies []containerProxyInfo
	for containerName, proxies := range containerProxyMap {
		containerProxies = append(containerProxies, containerProxyInfo{
			Name:    containerName,
			Proxies: proxies,
		})
	}

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
		tpProxies:       make(map[string]*toxiproxy.Proxy),
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/proxies", s.addProxyHandler).Methods("POST")
	r.HandleFunc("/api/proxies", s.getProxiesHandler).Methods("GET")
	r.HandleFunc("/api/proxies", s.deleteProxyHandler).Methods("DELETE")
	r.HandleFunc("/api/proxies/{proxyName}/toxics", s.createToxicHandler).Methods("POST")
	r.HandleFunc("/api/conns", s.getConnsHandler).Methods("GET")
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

func canonicalName(c dockerclient.Container) string {
	name := c.Names[0]
	for _, n := range c.Names {
		parts := strings.Split(n, "/")
		lastPart := parts[len(parts)-1]
		if len(lastPart) < len(name) {
			name = lastPart
		}
	}
	return name
}
