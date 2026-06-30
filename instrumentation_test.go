package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// failingResponseWriter is a ResponseWriter whose body writes always fail, used
// to drive the handlers' 5xx error paths.
type failingResponseWriter struct {
	header http.Header
}

func (w *failingResponseWriter) Header() http.Header { return w.header }

func (w *failingResponseWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

func (w *failingResponseWriter) WriteHeader(_ int) {}

func newRecordingProvider(t *testing.T) (*sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() { _ = provider.Shutdown(t.Context()) })

	return provider, exporter
}

func Test_recordServerError_setsErrorStatusWithMessage(t *testing.T) {
	provider, exporter := newRecordingProvider(t)

	ctx, span := provider.Tracer("test").Start(t.Context(), "test")
	recordServerError(ctx, errors.New("boom"))
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	if spans[0].Status.Code != codes.Error {
		t.Errorf("expected ERROR status, got %v", spans[0].Status.Code)
	}

	if spans[0].Status.Description != "boom" {
		t.Errorf("expected status message %q, got %q", "boom", spans[0].Status.Description)
	}
}

// Test_dataHandler_recordsErrorSpanOnWriteFailure exercises the 5xx path of a
// handler and asserts the active server span is marked ERROR with a non-empty
// diagnostic message, as required for error traces to be actionable.
func Test_dataHandler_recordsErrorSpanOnWriteFailure(t *testing.T) {
	provider, exporter := newRecordingProvider(t)

	ctx, span := provider.Tracer("test").Start(t.Context(), "GET /data")
	req := httptest.NewRequest(http.MethodGet, "/data?size=2048", http.NoBody).WithContext(ctx)

	dataHandler(&failingResponseWriter{header: http.Header{}}, req)
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	if spans[0].Status.Code != codes.Error {
		t.Errorf("expected ERROR status, got %v", spans[0].Status.Code)
	}

	if spans[0].Status.Description == "" {
		t.Error("expected a non-empty status message on the error span")
	}
}
