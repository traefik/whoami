package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Units.
const (
	_        = iota
	KB int64 = 1 << (10 * iota)
	MB
	GB
	TB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

var (
	cert    string
	key     string
	ca      string
	port    string
	name    string
	verbose bool
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.StringVar(&cert, "cert", "", "give me a certificate")
	flag.StringVar(&key, "key", "", "give me a key")
	flag.StringVar(&ca, "cacert", "", "give me a CA chain, enforces mutual TLS")
	flag.StringVar(&port, "port", getEnv("WHOAMI_PORT_NUMBER", "80"), "give me a port number")
	flag.StringVar(&name, "name", os.Getenv("WHOAMI_NAME"), "give me a name")
}

// Data whoami information.
type Data struct {
	Hostname   string            `json:"hostname,omitempty"`
	IP         []string          `json:"ip,omitempty"`
	Headers    http.Header       `json:"headers,omitempty"`
	URL        string            `json:"url,omitempty"`
	Host       string            `json:"host,omitempty"`
	Method     string            `json:"method,omitempty"`
	Name       string            `json:"name,omitempty"`
	RemoteAddr string            `json:"remoteAddr,omitempty"`
	Environ    map[string]string `json:"environ,omitempty"`
}

func main() {
	flag.Parse()

	mux := http.NewServeMux()
	mux.Handle("/data", handle(dataHandler, verbose))
	mux.Handle("/echo", handle(echoHandler, verbose))
	mux.Handle("/bench", handle(benchHandler, verbose))
	mux.Handle("/api", handle(apiHandler, verbose))
	mux.Handle("/health", handle(healthHandler, verbose))
	mux.Handle("/", handle(whoamiHandler, verbose))

	if cert == "" || key == "" {
		log.Printf("Starting up on port %s", port)

		log.Fatal(http.ListenAndServe(":"+port, mux))
	}

	server := &http.Server{
		Addr:      ":" + port,
		TLSConfig: &tls.Config{ClientAuth: tls.RequestClientCert},
		Handler:   mux,
	}

	if ca != "" {
		server.TLSConfig = setupMutualTLS(ca)
	}

	log.Printf("Starting up with TLS on port %s", port)

	log.Fatal(server.ListenAndServeTLS(cert, key))
}

func setupMutualTLS(ca string) *tls.Config {
	clientCACert, err := os.ReadFile(ca)
	if err != nil {
		log.Fatal(err)
	}

	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(clientCACert)

	tlsConfig := &tls.Config{
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                clientCertPool,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}

	return tlsConfig
}

func handle(next http.HandlerFunc, verbose bool) http.Handler {
	if !verbose {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next(w, r)

		// <remote_IP_address> - [<timestamp>] "<request_method> <request_path> <request_protocol>" -
		log.Printf("%s - - [%s] \"%s %s %s\" - -", r.RemoteAddr, time.Now().Format("02/Jan/2006:15:04:05 -0700"), r.Method, r.URL.Path, r.Proto)
	})
}

func benchHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprint(w, "1")
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

func printBinary(s []byte) {
	fmt.Printf("Received b:")
	for n := range s {
		fmt.Printf("%d,", s[n])
	}
	fmt.Printf("\n")
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	size, err := strconv.ParseInt(queryParams.Get("size"), 10, 64)
	if err != nil {
		size = 1
	}
	if size < 0 {
		size = 0
	}

	unit := queryParams.Get("unit")
	switch strings.ToLower(unit) {
	case "kb":
		size *= KB
	case "mb":
		size *= MB
	case "gb":
		size *= GB
	case "tb":
		size *= TB
	}

	attachment, err := strconv.ParseBool(queryParams.Get("attachment"))
	if err != nil {
		attachment = false
	}

	content := &contentReader{size: size}

	if attachment {
		w.Header().Set("Content-Disposition", "Attachment")
		http.ServeContent(w, r, "data.txt", time.Now(), content)
		return
	}

	if _, err := io.Copy(w, content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func whoamiHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	wait := queryParams.Get("wait")
	if wait != "" {
		duration, err := time.ParseDuration(wait)
		if err == nil {
			time.Sleep(duration)
		}
	}

	if name != "" {
		_, _ = fmt.Fprintln(w, "Name:", name)
	}

	hostname, _ := os.Hostname()
	_, _ = fmt.Fprintln(w, "Hostname:", hostname)

	for _, ip := range getIPs() {
		_, _ = fmt.Fprintln(w, "IP:", ip)
	}

	_, _ = fmt.Fprintln(w, "RemoteAddr:", r.RemoteAddr)

	if r.TLS != nil {
		for i, cert := range r.TLS.PeerCertificates {
			_, _ = fmt.Fprintf(w, "Certificate[%d] Subject: %v\n", i, cert.Subject)
		}
	}

	if err := r.Write(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if ok, _ := strconv.ParseBool(queryParams.Get("env")); ok {
		for _, env := range os.Environ() {
			_, _ = fmt.Fprintln(w, env)
		}
	}
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	hostname, _ := os.Hostname()

	environ := make(map[string]string)

	if ok, _ := strconv.ParseBool(queryParams.Get("env")); ok {
		for _, env := range os.Environ() {
			before, after, _ := strings.Cut(env, "=")
			environ[before] = after
		}
	}

	data := Data{
		Hostname:   hostname,
		IP:         getIPs(),
		Headers:    r.Header,
		URL:        r.URL.RequestURI(),
		Host:       r.Host,
		Method:     r.Method,
		Name:       name,
		RemoteAddr: r.RemoteAddr,
		Environ:    environ,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type healthState struct {
	StatusCode int
}

var (
	currentHealthState = healthState{http.StatusOK}
	mutexHealthState   = &sync.RWMutex{}
)

func healthHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost {
		var statusCode int

		if err := json.NewDecoder(req.Body).Decode(&statusCode); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Printf("Update health check status code [%d]\n", statusCode)

		mutexHealthState.Lock()
		defer mutexHealthState.Unlock()
		currentHealthState.StatusCode = statusCode
	} else {
		mutexHealthState.RLock()
		defer mutexHealthState.RUnlock()
		w.WriteHeader(currentHealthState.StatusCode)
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getIPs() []string {
	var ips []string

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
			if ip != nil {
				ips = append(ips, ip.String())
			}
		}
	}

	return ips
}
