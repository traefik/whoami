package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	grpcWhoami "github.com/traefik/whoami/grpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	otellog "go.opentelemetry.io/otel/log"
	"google.golang.org/grpc"
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

// version is the whoami build, reported as the service.version resource
// attribute. Override it at build time with -ldflags "-X main.version=...".
var version = "dev"

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
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "whoami terminated:", err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownOTel, err := setupOTel(ctx)
	if err != nil {
		return fmt.Errorf("setting up OpenTelemetry: %w", err)
	}
	defer shutdownOTelWithTimeout(shutdownOTel)

	server := &http.Server{Addr: ":" + port, Handler: newMux()}

	if cert == "" || key == "" {
		// Accept HTTP/1.1 and HTTP/2 cleartext (h2c) so the gRPC endpoint keeps
		// working without TLS, using the native net/http support added in Go 1.24.
		protocols := new(http.Protocols)
		protocols.SetHTTP1(true)
		protocols.SetUnencryptedHTTP2(true)
		server.Protocols = protocols

		logInfo(ctx, "Starting up", otellog.String("port", port))

		return startServer(ctx, server, server.ListenAndServe)
	}

	server.TLSConfig = &tls.Config{ClientAuth: tls.RequestClientCert}
	if ca != "" {
		server.TLSConfig, err = setupMutualTLS(ca)
		if err != nil {
			return err
		}
	}

	logInfo(ctx, "Starting up with TLS", otellog.String("port", port))

	return startServer(ctx, server, func() error { return server.ListenAndServeTLS(cert, key) })
}

// newMux builds the whoami request router with every HTTP route instrumented for
// OpenTelemetry tracing, metrics, and access logging, plus the gRPC endpoint.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/data", instrument("/data", dataHandler))
	mux.Handle("/echo", accessLog(http.HandlerFunc(echoHandler)))
	mux.Handle("/bench", instrument("/bench", benchHandler))
	mux.Handle("/api", instrument("/api", apiHandler))
	mux.Handle("/health", instrument("/health", healthHandler))
	mux.Handle("/", instrument("/", whoamiHandler))

	serverGRPC := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	grpcWhoami.RegisterWhoamiServer(serverGRPC, whoamiServer{})
	mux.Handle("/whoami.Whoami/", serverGRPC)

	return mux
}

// startServer runs listen until it fails or the context is canceled (SIGINT or
// SIGTERM), then drains in-flight requests so the deferred telemetry flush runs.
func startServer(ctx context.Context, server *http.Server, listen func() error) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- listen()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("serving: %w", err)
	case <-ctx.Done():
		// The signal already canceled ctx, so derive a fresh deadline that keeps
		// ctx's values but not its cancellation to actually drain connections.
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()

		logInfo(shutdownCtx, "Shutting down")

		return server.Shutdown(shutdownCtx)
	}
}

// shutdownOTelWithTimeout flushes and stops the telemetry providers, bounding the
// flush so a stuck exporter cannot hang shutdown.
func shutdownOTelWithTimeout(shutdown func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := shutdown(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error during OpenTelemetry shutdown:", err)
	}
}

func setupMutualTLS(ca string) (*tls.Config, error) {
	clientCACert, err := os.ReadFile(ca)
	if err != nil {
		return nil, fmt.Errorf("reading CA chain: %w", err)
	}

	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(clientCACert)

	tlsConfig := &tls.Config{
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                clientCertPool,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}

	return tlsConfig, nil
}

func benchHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprint(w, "1")
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logError(r.Context(), "WebSocket upgrade failed", otellog.String("error", err.Error()))
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
		recordServerError(r.Context(), err)
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
		recordServerError(r.Context(), err)
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
		recordServerError(r.Context(), err)
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

type whoamiServer struct {
	grpcWhoami.UnimplementedWhoamiServer
}

func (g whoamiServer) Bench(_ context.Context, _ *grpcWhoami.BenchRequest) (*grpcWhoami.BenchReply, error) {
	return &grpcWhoami.BenchReply{Data: 1}, nil
}

func (g whoamiServer) Whoami(_ context.Context, _ *grpcWhoami.WhoamiRequest) (*grpcWhoami.WhoamiReply, error) {
	reply := &grpcWhoami.WhoamiReply{}
	if name != "" {
		reply.Name = name
	}

	reply.Hostname, _ = os.Hostname()

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
			reply.Iface = append(reply.Iface, ip.String())
		}
	}

	return reply, nil
}
