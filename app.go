package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"os"
	"net"
)

func main() {

	r := mux.NewRouter()
	r.HandleFunc("/", whoamI)
	http.Handle("/", r)
	fmt.Println("Starting up on 80")
	log.Fatal(http.ListenAndServe(":80", nil))
}

func whoamI(w http.ResponseWriter, req *http.Request) {
	hostname, _ := os.Hostname()
	fmt.Fprintln(w, "Hostname : ", hostname)
	ifaces, _ := net.Interfaces()
	// handle err
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
					fmt.Fprintln(w, "IP : ", ip)
	    }
	}
  req.Write(w)
}
