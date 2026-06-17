package main

import (
	"bufio"
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
)

// instrument wraps an HTTP handler with OpenTelemetry tracing and metrics and
// structured access logging. The span is started first so that each access-log
// record is correlated with the active trace and span.
func instrument(route string, handler http.HandlerFunc) http.Handler {
	withLog := accessLog(handler)

	return otelhttp.NewHandler(withLog, route,
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + route
		}),
	)
}

// accessLog increments the request counter metric and, when verbose logging is
// enabled, records a structured access-log entry per request. Gating the log on
// the verbose flag keeps whoami quiet by default, matching its original
// behavior. The handler runs inside the OpenTelemetry span, so the access log
// carries the trace and span IDs, tying logs, traces, and metrics together.
func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		requestCounter.Add(r.Context(), 1, metric.WithAttributes(
			attribute.String("http.request.method", r.Method),
			attribute.Int("http.response.status_code", recorder.status),
		))

		if !verbose {
			return
		}

		logInfo(r.Context(), "access",
			otellog.String("network.peer.address", r.RemoteAddr),
			otellog.String("http.request.method", r.Method),
			otellog.String("url.path", r.URL.Path),
			otellog.String("network.protocol.name", r.Proto),
			otellog.Int("http.response.status_code", recorder.status),
			otellog.Int64("http.response.body.size", recorder.written),
			otellog.Float64("http.server.request.duration", time.Since(start).Seconds()),
		)
	})
}

// responseRecorder captures the response status code and body size while
// forwarding the optional interfaces (Hijacker, Flusher) that h2c and WebSocket
// upgrades depend on.
type responseRecorder struct {
	http.ResponseWriter

	status      int
	written     int64
	wroteHeader bool
}

// WriteHeader records the first status code written to the response.
func (r *responseRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
	}

	r.ResponseWriter.WriteHeader(status)
}

// Write tracks the number of bytes written to the response body.
func (r *responseRecorder) Write(b []byte) (int, error) {
	r.wroteHeader = true

	n, err := r.ResponseWriter.Write(b)
	r.written += int64(n)

	return n, err
}

// Hijack lets WebSocket upgrades take over the underlying connection.
func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(r.ResponseWriter).Hijack()
}

// Flush forwards flushes to the underlying response writer.
func (r *responseRecorder) Flush() {
	_ = http.NewResponseController(r.ResponseWriter).Flush()
}

// Unwrap exposes the wrapped response writer to http.ResponseController.
func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
