package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

var cert string
var key string
var port string

const (
	_           = iota
	KB int64 = 1 << (10 * iota)
	MB
	GB
	TB
)

func init() {
	flag.StringVar(&cert, "cert", "", "give me a certificate")
	flag.StringVar(&key, "key", "", "give me a key")
	flag.StringVar(&port, "port", "80", "give me a port number")
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	flag.Parse()
	http.HandleFunc("/data", dataHandler)
	http.HandleFunc("/echo", echoHandler)
	http.HandleFunc("/bench", benchHandler)
	http.HandleFunc("/", whoami)
	http.HandleFunc("/api", api)
	http.HandleFunc("/health", healthHandler)
	fmt.Println("Starting up on port " + port)
	if len(cert) > 0 && len(key) > 0 {
		log.Fatal(http.ListenAndServeTLS(":"+port, cert, key, nil))
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func printBinary(s []byte) {
	fmt.Printf("Received b:")
	for n := 0; n < len(s); n++ {
		fmt.Printf("%d,", s[n])
	}
	fmt.Printf("\n")
}
func benchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, "1")
}
func echoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			return
		}
		printBinary(p)
		err = conn.WriteMessage(messageType, p)
		if err != nil {
			return
		}
	}
}
func dataHandler(w http.ResponseWriter, r *http.Request) {
	u, _ := url.Parse(r.URL.String())
	queryParams := u.Query()

	size, err := strconv.ParseInt(queryParams.Get("size"), 10, 64)
	if err != nil {
		size = 1
	}
	if size < 0 {
		size = 0
	}

	unit := queryParams.Get("unit")
	switch (strings.ToLower(unit)) {
		case "kb": size = size * KB
		case "mb": size = size * MB
		case "gb": size = size * GB
		case "tb": size = size * TB
	}

	attachment, err := strconv.ParseBool(queryParams.Get("attachment"))
	if err != nil {
		attachment = false
	}

	content := fillContent(size)

	if attachment {
		const name = "data.txt"
		w.Header().Add("Content-Disposition", "Attachment")
		http.ServeContent(w, r, name, time.Now(), content)
	} else {
		io.Copy(w, content)
	}
}
func whoami(w http.ResponseWriter, req *http.Request) {
	u, _ := url.Parse(req.URL.String())
	queryParams := u.Query()
	wait := queryParams.Get("wait")
	if len(wait) > 0 {
		duration, err := time.ParseDuration(wait)
		if err == nil {
			time.Sleep(duration)
		}
	}
	hostname, _ := os.Hostname()
	fmt.Fprintln(w, "Hostname:", hostname)
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			fmt.Fprintln(w, "IP:", ip)
		}
	}
	req.Write(w)
}

func api(w http.ResponseWriter, req *http.Request) {
	hostname, _ := os.Hostname()
	data := struct {
		Hostname string      `json:"hostname,omitempty"`
		IP       []string    `json:"ip,omitempty"`
		Headers  http.Header `json:"headers,omitempty"`
		URL      string      `json:"url,omitempty"`
		Host     string      `json:"host,omitempty"`
		Method   string      `json:"method,omitempty"`
	}{
		hostname,
		[]string{},
		req.Header,
		req.URL.RequestURI(),
		req.Host,
		req.Method,
	}

	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			data.IP = append(data.IP, ip.String())
		}
	}
	json.NewEncoder(w).Encode(data)
}

type healthState struct {
	StatusCode int
}

var currentHealthState = healthState{200}
var mutexHealthState = &sync.RWMutex{}

func healthHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost {
		var statusCode int
		err := json.NewDecoder(req.Body).Decode(&statusCode)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
		} else {
			fmt.Printf("Update health check status code [%d]\n", statusCode)
			mutexHealthState.Lock()
			defer mutexHealthState.Unlock()
			currentHealthState.StatusCode = statusCode
		}
	} else {
		mutexHealthState.RLock()
		defer mutexHealthState.RUnlock()
		w.WriteHeader(currentHealthState.StatusCode)
	}
}

func fillContent(length int64) io.ReadSeeker {
	charset := "-ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length, length)

	for i := range b {
		b[i] = charset[i % len(charset)]
	}

	if length > 0 {
		b[0] = '|'
		b[length-1] = '|'
	}

	return bytes.NewReader(b)
}