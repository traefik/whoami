package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	http.HandleFunc("/echo", echoHandler)
	http.HandleFunc("/", whoamI)
	http.HandleFunc("/api", api)
	fmt.Println("Starting up on 80")
	log.Fatal(http.ListenAndServe(":80", nil))
}

func print_binary(s []byte) {
	fmt.Printf("Received b:")
	for n := 0; n < len(s); n++ {
		fmt.Printf("%d,", s[n])
	}
	fmt.Printf("\n")
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
		print_binary(p)
		err = conn.WriteMessage(messageType, p)
		if err != nil {
			return
		}
	}
}

func whoamI(w http.ResponseWriter, req *http.Request) {
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
		Hostname string   `json:"hostname,omitempty"`
		IP       []string `json:"ip,omitempty"`
	}{
		hostname,
		[]string{},
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
